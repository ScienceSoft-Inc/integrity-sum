VERSION_MOCKGEN=v1.6.0
## You can change these values
RELEASE_NAME_DB=db
RELEASE_NAME_APP=app

GIT_COMMIT := $(shell git describe --tags --long --dirty=-unsupported --always || echo pre-commit)
IMAGE_VERSION ?= $(GIT_COMMIT)

BINARY_NAME=integritySum
IMAGE_NAME?=integrity
FULL_IMAGE_NAME=$(IMAGE_NAME):$(IMAGE_VERSION)

# helm chart path
HELM_CHART_DB = helm-charts/database-to-integrity-sum
HELM_CHART_APP = helm-charts/app-to-monitor

DB_SECRET_NAME ?= $(IMAGE_NAME)

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
.PHONY : test
test: generate
	go test -v ./internal/core/services/hash_test.go
	go test -v ./pkg/hasher

## Downloads the necessesary dev dependencies.
.PHONY : dev-dependencies
dev-dependencies: minikube update docker helm-all
	@echo "==> Downloaded development dependencies"
	@echo "==> Successfully installed"

.PHONY : minikube
minikube:
	minikube start

.PHONY : docker
docker:
	@eval $$(minikube docker-env) ;\
    docker image build -t $(FULL_IMAGE_NAME) -t $(IMAGE_NAME):latest .

.PHONY : helm-all
helm-all: helm-database helm-app

helm-database:
	# @helm dependency update $(HELM_CHART_DB)
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
		$(HELM_CHART_APP)

.PHONY: kind-load-images
kind-load-images:
	kind load docker-image hasher:latest
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

.PHONY : build
build:
	go build -o ${BINARY_NAME} ./cmd/k8s-integrity-sum

.PHONY : run
run: build
	./${BINARY_NAME}

## Remove an installed Helm deployment and stop minikube.
stop:
	helm uninstall  ${RELEASE_NAME_APP}
	helm uninstall  ${RELEASE_NAME_DB}
	minikube stop

## Cleaning build cache.
.PHONY : clean
clean:
	go clean
	rm ${BINARY_NAME}

## Compile the binary.
compile-all: windows-32bit windows-64bit linux-32bit linux-64bit MacOS

windows-32bit:
	echo "Building for Windows 32-bit"
	GOOS=windows GOARCH=386 go build -o bin/${BINARY_NAME}_32bit.exe cmd/k8s-integrity-sum/main.go

windows-64bit:
	echo "Building for Windows 64-bit"
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}_64bit.exe cmd/k8s-integrity-sum/main.go

linux-32bit:
	echo "Building for Linux 32-bit"
	GOOS=linux GOARCH=386 go build -o bin/${BINARY_NAME}_32bit cmd/k8s-integrity-sum/main.go

linux-64bit:
	echo "Building for Linux 64-bit"
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME} cmd/k8s-integrity-sum/main.go

MacOS:
	echo "Building for MacOS X 64-bit"
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}_macos cmd/k8s-integrity-sum/main.go

# build bee2 library
.PHONY: bee2-lib
bee2-lib:
	@mkdir -p bee2/build
	@cmake -S bee2 -B bee2/build
	@make -C bee2/build
