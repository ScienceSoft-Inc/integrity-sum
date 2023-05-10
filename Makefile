VERSION_MOCKGEN := v1.6.0
## You can change these values
RELEASE_NAME_APP    := app
RELEASE_NAME_SYSLOG := rsyslog

# build configuration options
SYSLOG_ENABLED  ?= false
BEE2_ENABLED    ?= false
DURATION_TIME   ?= 25s
MINIO_ENABLED   ?= false

GIT_COMMIT      := $(shell git describe --tags --long --dirty=-unsupported --always || echo pre-commit)
IMAGE_VERSION   ?= $(GIT_COMMIT)

BIN             := bin
ENVTEST         := $(CUR_DIR)/$(BIN)/setup-envtest
GINKGO          := $(CUR_DIR)/$(BIN)/ginkgo
BINARY          := $(BIN)/integritySum
IMAGE_NAME      ?= integrity
FULL_IMAGE_NAME := $(IMAGE_NAME):$(IMAGE_VERSION)

# build options
CUR_DIR     := $(shell pwd)
REPO        := github.com/ScienceSoft-Inc/integrity-sum
SRCROOT     := /go/src/$(REPO)
SRC_APP     := cmd/k8s-integrity-sum
PACKAGE_APP := $(REPO)/$(SRC_APP)

BUILDTOOL_IMAGE := buildtools:latest
DOCKER_RUNNER   := docker run --rm -v $(CUR_DIR):$(SRCROOT) -w $(SRCROOT) -u `id -u`:`id -g`
GO_CACHE        := $(BIN)/go-cache
GO_BUILD_FLAGS  ?= -pkgdir $(SRCROOT)/$(GO_CACHE) -v -ldflags "-linkmode external -extldflags '-static' -s -w"
CCBUILD         := $(DOCKER_RUNNER) $(BUILDTOOL_IMAGE) gcc
CMAKETOOL       := $(DOCKER_RUNNER) $(BUILDTOOL_IMAGE) cmake
MAKETOOL        := $(DOCKER_RUNNER) $(BUILDTOOL_IMAGE) make
BUILDER         := $(DOCKER_RUNNER) -e CGO_ENABLED=1 -e GO111MODULE=off -e GOCACHE=$(SRCROOT)/$(GO_CACHE) $(BUILDTOOL_IMAGE)
GOBUILD         := $(BUILDER) go build $(GO_BUILD_FLAGS)
GOTEST          := $(BUILDER) go test -v

# helm chart path
HELM_CHART_APP      := helm-charts/app-to-monitor
HELM_CHART_SYSLOG   := helm-charts/rsyslog

#
EMPTY :=
SPACE := $(EMPTY) $(EMPTY)
COMMA := ,

join-with = $(subst $(SPACE),$1,$(strip $2))

# tags defining
ifeq ($(BEE2_ENABLED), true)
TAGLIST += bee2
endif

# TAGLIST += test

TAGS_JOINED := $(call join-with,$(COMMA),$(TAGLIST))
TAGS        := $(if $(strip $(TAGS_JOINED)),-tags $(TAGS_JOINED),)

## Runs the test suite with mocks enabled.
.PHONY: test
test: test-bee2 tminio
	@$(GOTEST) -timeout 5s \
	 	./pkg/hasher \
		./internal/walker \
		./internal/worker

.PHONY: test-bee2
test-bee2:
ifeq ($(BEE2_ENABLED), true)
	@$(GOTEST) -tags bee2 ./internal/ffi/bee2
endif

.PHONY: tminio
tminio:
	go test -v -timeout 20s ./pkg/minio

## Downloads the necessesary dev dependencies.
.PHONY : dev-dependencies
dev-dependencies: minikube update docker helm-all
	@echo "==> Downloaded development dependencies"
	@echo "==> Successfully installed"

.PHONY : minikube
minikube:
	minikube start

# Downloads the Go module.
.PHONY : vendor
vendor:
	@echo "==> Downloading vendor"
	go mod tidy
	go mod vendor

.PHONY: docker
docker: vendor
	@docker build -t $(FULL_IMAGE_NAME) -t $(IMAGE_NAME):latest -f ./docker/Dockerfile .

.PHONY : helm-all
helm-all: helm-app
helm-app:
	@helm upgrade -i ${RELEASE_NAME_APP} \
		--set containerSidecar.name=$(IMAGE_NAME) \
		--set containerSidecar.image=$(FULL_IMAGE_NAME) \
		--set configMap.syslog.enabled=$(SYSLOG_ENABLED) \
		--set configMap.durationTime=$(DURATION_TIME) \
		--set minio.enabled=$(MINIO_ENABLED) \
		$(HELM_CHART_APP) --wait

helm-syslog:
	@helm upgrade -i ${RELEASE_NAME_SYSLOG} \
		$(HELM_CHART_SYSLOG) --wait

.PHONY: kind-load-images
kind-load-images:
	@kind load docker-image $(FULL_IMAGE_NAME) controller:latest

.PHONY : tidy
tidy: ## Cleans the Go module.
	@echo "==> Tidying module"
	@go mod tidy

.PHONY: build
build: dirs
	@$(GOBUILD) -o ${BINARY} $(TAGS) $(PACKAGE_APP)

.PHONY : run
run: build
	./${BINARY}

## Remove an installed Helm deployment and stop minikube.
stop:
	helm uninstall ${RELEASE_NAME_APP}
	minikube stop

## Cleaning build cache.
.PHONY : clean
clean:
	go clean
	rm ${BINARY}

## Compile the binary.
compile-all: windows-32bit windows-64bit linux-32bit linux-64bit MacOS

windows-32bit:
	echo "Building for Windows 32-bit"
	GOOS=windows GOARCH=386 go build $(TAGS) -o ${BINARY}_32bit.exe ./$(SRC_APP)

windows-64bit:
	echo "Building for Windows 64-bit"
	GOOS=windows GOARCH=amd64 go build $(TAGS) -o ${BINARY}_64bit.exe ./$(SRC_APP)

linux-32bit:
	echo "Building for Linux 32-bit"
	GOOS=linux GOARCH=386 go build $(TAGS) -o ${BINARY}_32bit ./$(SRC_APP)

linux-64bit:
	echo "Building for Linux 64-bit"
	GOOS=linux GOARCH=amd64 go build $(TAGS) -o ${BINARY} ./$(SRC_APP)

MacOS:
	echo "Building for MacOS X 64-bit"
	GOOS=darwin GOARCH=amd64 go build $(TAGS) -o ${BINARY}_macos ./$(SRC_APP)

# build bee2 library within container
.PHONY: bee2-lib
bee2-lib:
	@$(CMAKETOOL) -S ./bee2 -B ./bee2/build
	@$(MAKETOOL) -C ./bee2/build

.PHONY: dirs
dirs:
	@mkdir -p bee2/build $(BIN)

.PHONY: buildtools
buildtools:
	@docker build -f ./docker/Dockerfile.build -t buildtools:latest ./docker


# Take snapshot of a docker image file system.
#
# Usage example:
#
# 	$ IMAGE_EXPORT=integrity:latest make export-fs
# ..will export the filesystem of the image "integrity:latest" into the predefined
# direcrory.
#
# 	$ ALG=SHA512 DIRS="app,bin" make snapshot
# ..will create snapshot for the "app" and "bin" directories of the exported early
# file system using the SHA512 algorithm.

ALG ?= sha256
ALG := $(shell echo $(ALG) | tr '[:upper:]' '[:lower:]')

DOCKER_FS_DIR := $(BIN)/docker-fs
SNAPSHOT_DIR  := helm-charts/snapshot/files

ifneq (,$(IMAGE_EXPORT))
  SNAPSHOT_OUTPUT := $(SNAPSHOT_DIR)/$(IMAGE_EXPORT).$(ALG)
else
  SNAPSHOT_OUTPUT := $(SNAPSHOT_DIR)/snapshot.$(ALG)
endif

ifeq (export-fs,$(firstword $(MAKECMDGOALS)))
  CID:=$(shell docker create $(IMAGE_EXPORT))
endif

.PHONY: export-fs
export-fs: ensure-export-dir clear-snapshots
	@docker export $(CID) | tar -xC $(DOCKER_FS_DIR) && docker rm $(CID) > /dev/null 2>&1 && \
	echo exported to $(DOCKER_FS_DIR)

.PHONY: snapshot
snapshot: ensure-snapshot-dir
	@go run ./cmd/snapshot --root-fs="$(DOCKER_FS_DIR)" --dir '$(DIRS)' --algorithm $(ALG) --out $(SNAPSHOT_OUTPUT) && \
	echo created $(SNAPSHOT_OUTPUT) && \
	cat $(SNAPSHOT_OUTPUT)

.PHONY: ensure-export-dir
ensure-export-dir:
	@mkdir -p $(DOCKER_FS_DIR)

.PHONY: ensure-snapshot-dir
ensure-snapshot-dir:
	@mkdir -p $(SNAPSHOT_DIR)

.PHONY: clear-snapshots
clear-snapshots:
	@-rm -rf $(DOCKER_FS_DIR)/* $(SNAPSHOT_DIR)/*

RELEASE_NAME_SNAPSHOT   ?= snapshot-crd
HELM_CHART_SNAPSHOT     := helm-charts/snapshot

.PHONY: helm-snapshot
helm-snapshot:
	@helm upgrade -i $(RELEASE_NAME_SNAPSHOT) $(HELM_CHART_SNAPSHOT)

# Create and install snapshot CRD with controller

CRD_MAKE := make -C snapshot-controller

.PHONY: crd-controller-build
crd-controller-build:
	$(CRD_MAKE) docker-build

.PHONY: crd-controller-deploy
crd-controller-deploy:
	$(CRD_MAKE) install deploy

.PHONY: crd-controller-test
crd-controller-test:
	$(CRD_MAKE) manifests generate install test

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(BIN)
	test -s $(BIN)/setup-envtest || GOBIN=$(CUR_DIR)/$(BIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO): $(BIN)
	test -s $(BIN)/ginkgo || GOBIN=$(CUR_DIR)/$(BIN) go install github.com/onsi/ginkgo/v2/ginkgo@2.9.2

.PHONY: minio-install
minio-install:
	echo "Install Minio server"
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm repo update
	helm upgrade -i minio --namespace=minio bitnami/minio --create-namespace \
 		--set defaultBuckets=integrity \
 		--wait

.PHONY: load-images
load-images:
	minikube image load $(FULL_IMAGE_NAME) controller:latest

.PHONY: install-test-cr
install-test-cr:
	@helm upgrade -i test-cr e2e/chart/snapshot --wait

.PHONY: e2etest
e2etest: install-test-cr envtest ginkgo
	make helm-app SYSLOG_ENABLED=true DURATION_TIME=2s MINIO_ENABLED=true
	$(BIN)/ginkgo -v ./e2e/...
