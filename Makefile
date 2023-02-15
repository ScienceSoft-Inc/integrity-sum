BINARY_NAME=integritySum
VERSION_MOCKGEN=v1.6.0
## You can change these values
RELEASE_NAME_DB=db
RELEASE_NAME_APP=app

# helm chart path
HELM_CHART_DB = helm-charts/database-to-integrity-sum
HELM_CHART_APP = helm-charts/app-to-monitor

DB_SECRET_NAME ?= secret-database-to-integrity-sum

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
	@echo "==> Generated a mocks"

## Runs the test suite with mocks enabled.
.PHONY : test
test: generate
	go test -v internal/core/services/hash_test.go
	cd pkg/hasher ;\
	go test

## Downloads the necessesary dev dependencies.
.PHONY : dev-dependencies
dev-dependencies: minikube update docker helm-all
	@echo "==> Downloaded development dependencies"
	@echo "==> Successfully installed"

.PHONY : minikube
minikube:
	minikube start

# SECRET_DB="$$(grep 'secretName' $(HELM_CHART_DB)/values.yaml | cut -d':' -f2 | tr -d '[:space:]')"
# SECRET_HASHER="$$(grep 'secretNameDB' helm-charts/app-to-monitor/values.yaml | cut -d':' -f2 | tr -d '[:space:]')"
# VALUE_RELEASE_NAME_APP="$$(grep 'releaseNameDB' helm-charts/app-to-monitor/values.yaml | cut -d':' -f2 | tr -d '[:space:]')"
# .PHONY: update
# update:
# 	sed -i "s/${SECRET_HASHER}/${SECRET_DB}/" helm-charts/app-to-monitor/values.yaml >> helm-charts/app-to-monitor/values.yaml
# 	sed -i "s/${VALUE_RELEASE_NAME_APP}/${RELEASE_NAME_DB}/" helm-charts/app-to-monitor/values.yaml >> helm-charts/app-to-monitor/values.yaml

.PHONY : docker
docker:
	@eval $$(minikube docker-env) ;\
    docker image build -t hasher:latest -f Dockerfile .

.PHONY : helm-all
helm-all: helm-database helm-app

helm-database:
	@helm dependency update $(HELM_CHART_DB)
	@helm upgrade -i ${RELEASE_NAME_DB} \
		--set postgresql.auth.database=$(DB_NAME) \
		--set postgresql.auth.username=$(DB_USER) \
		--set postgresql.auth.password=$(DB_PASSWORD) \
		--set-string secretName=$(DB_SECRET_NAME) \
		$(HELM_CHART_DB)

helm-app:
	@helm upgrade -i ${RELEASE_NAME_APP} \
		--set releaseNameDB=$(DB_NAME) \
		--set secretNameDB=$(DB_SECRET_NAME) \
		$(HELM_CHART_APP)

.PHONY: kind-load-images
kind-load-images:
	kind load docker-image hasher:latest 

.PHONY : tidy
tidy: ## Cleans the Go module.
	@echo "==> Tidying module"
	@go mod tidy

.PHONY : build
build:
	go build -o ${BINARY_NAME} cmd/k8s-integrity-sum/main.go

.PHONY : run
run:
	go build -o ${BINARY_NAME} cmd/k8s-integrity-sum/main.go
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
