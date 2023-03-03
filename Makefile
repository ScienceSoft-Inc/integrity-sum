VERSION_MOCKGEN := v1.6.0
## You can change these values
RELEASE_NAME_DB     ?= db
RELEASE_NAME_APP    := app
RELEASE_NAME_SYSLOG := rsyslog

# build configuration options
SYSLOG_ENABLED  ?= false
BEE2_ENABLED    ?= false

GIT_COMMIT      := $(shell git describe --tags --long --dirty=-unsupported --always || echo pre-commit)
IMAGE_VERSION   ?= $(GIT_COMMIT)

BIN             := bin
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
HELM_CHART_DB       := helm-charts/database-to-integrity-sum
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

## Runs all of the required cleaning and verification targets.
.PHONY : all
all: mod-download test dev-dependencies

## Downloads the Go module.
.PHONY : mod-download
mod-download:
	@echo "==> Downloading Go module"
	go mod download

## Generates folders with mock functions
.PHONY : generate
generate:
	go install github.com/golang/mock/mockgen@${VERSION_MOCKGEN}
	export PATH=$PATH:$(go env GOPATH)/bin
	go generate ./internal/core/ports/repository.go
	go generate ./internal/core/ports/service.go
	@echo "==> Mocks have been generated"

## Runs the test suite with mocks enabled.
.PHONY: test
test: generate test-bee2
	@$(GOTEST) ./internal/core/services \
	 	./pkg/hasher

.PHONY: test-bee2
test-bee2:
ifeq ($(BEE2_ENABLED), true)
	@$(GOTEST) -tags bee2 ./internal/ffi/bee2
endif

## Downloads the necessesary dev dependencies.
.PHONY : dev-dependencies
dev-dependencies: minikube update docker helm-all
	@echo "==> Downloaded development dependencies"
	@echo "==> Successfully installed"

.PHONY : minikube
minikube:
	minikube start

.PHONY: docker
docker:
	@docker build -t $(FULL_IMAGE_NAME) -t $(IMAGE_NAME):latest -f ./docker/Dockerfile .

.PHONY : helm-all
helm-all: helm-database helm-app

helm-database:
	@helm upgrade -i ${RELEASE_NAME_DB} \
		--set global.postgresql.auth.database=$(DB_NAME) \
		--set global.postgresql.auth.username=$(DB_USER) \
		--set global.postgresql.auth.password=$(DB_PASSWORD) \
		$(HELM_CHART_DB)

helm-app:
	@helm upgrade -i ${RELEASE_NAME_APP} \
		--set releaseNameDB=$(RELEASE_NAME_DB) \
		--set db.database=$(DB_NAME) \
		--set db.username=$(DB_USER) \
		--set db.password=$(DB_PASSWORD) \
		--set containerSidecar.name=$(IMAGE_NAME) \
		--set containerSidecar.image=$(FULL_IMAGE_NAME) \
		--set configMap.syslog.enabled=$(SYSLOG_ENABLED) \
		$(HELM_CHART_APP)

helm-syslog:
	@helm upgrade -i ${RELEASE_NAME_SYSLOG} \
		$(HELM_CHART_SYSLOG)

.PHONY: kind-load-images
kind-load-images:
	@kind load docker-image $(FULL_IMAGE_NAME)

DB_PVC_NAME=$(shell kubectl get pvc | grep data-$(RELEASE_NAME_DB)-postgresql | cut -d " " -f1)

.PHONY: purge-db uninstall-db
purge-db: uninstall-db
	@-kubectl delete pvc $(DB_PVC_NAME)

uninstall-db:
	@-helm uninstall ${RELEASE_NAME_DB}

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
	helm uninstall ${RELEASE_NAME_DB}
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
