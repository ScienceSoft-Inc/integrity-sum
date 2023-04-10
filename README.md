# integrity-sum

[![GitHub contributors](https://img.shields.io/github/contributors/ScienceSoft-Inc/integrity-sum)](https://github.com/ScienceSoft-Inc/integrity-sum)
[![GitHub last commit](https://img.shields.io/github/last-commit/ScienceSoft-Inc/integrity-sum)](https://github.com/ScienceSoft-Inc/integrity-sum)
[![GitHub](https://img.shields.io/github/license/ScienceSoft-Inc/integrity-sum)](https://github.com/ScienceSoft-Inc/integrity-sum/blob/main/LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/ScienceSoft-Inc/integrity-sum)](https://github.com/ScienceSoft-Inc/integrity-sum/issues)
[![GitHub forks](https://img.shields.io/github/forks/ScienceSoft-Inc/integrity-sum)](https://github.com/ScienceSoft-Inc/integrity-sum/network/members)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![GitHub](https://img.shields.io/badge/github-%23121011.svg?style=for-the-badge&logo=github&logoColor=white)

This program provides integrity monitoring that checks file or directory of container to determine whether or not they have been tampered with or corrupted.
integrity-sum, which is a type of change auditing, verifies and validates these files by comparing them to the stored data in the database.

If program detects that files have been altered, updated, added or compromised, it rolls back deployments to a previous version.

integrity-sum injects a `hasher-sidecar` to your pods as a sidecar container.
`hasher-sidecar` the implementation of a hasher in golang, which calculates the checksum of files using different algorithms in kubernetes:

* MD5
* SHA256
* SHA1
* SHA224
* SHA384
* SHA512
* BEE2 (optional)

## Table of Contents

* [integrity-sum](#integrity-sum)
  * [Table of Contents](#table-of-contents)
  * [Architecture](#architecture)
    * [Statechart diagram](#statechart-diagram)
  * [Getting Started](#getting-started)
    * [Clone repository and install dependencies](#clone-repository-and-install-dependencies)
  * [Demo-App](#demo-app)
  * [:hammer: Installing components](#hammer-installing-components)
    * [Running locally](#running-locally)
    * [Install Helm](#install-helm)
    * [Configuration](#configuration)
  * [Quick start](#quick-start)
    * [Using Makefile](#using-makefile)
    * [Manual start](#manual-start)
  * [Pay attention](#pay-attention)
  * [Troubleshooting](#troubleshooting)
  * [:notebook\_with\_decorative\_cover: Godoc extracts and generates documentation for Go programs](#notebook_with_decorative_cover-godoc-extracts-and-generates-documentation-for-go-programs)
    * [Presents the documentation as a web page](#presents-the-documentation-as-a-web-page)
  * [:mag: Running tests](#mag-running-tests)
  * [:mag: Running linter "golangci-lint"](#mag-running-linter-golangci-lint)
  * [Including Bee2 library into the application](#including-bee2-library-into-the-application)
  * [Enable MinIO](#enable-minio)
    * [Install standalone server](#install-standalone-server)
    * [Include into the project](#include-into-the-project)
  * [Syslog support](#syslog-support)
    * [Install syslog server](#install-syslog-server)
    * [Syslog messages format](#syslog-messages-format)
  * [Creating a snapshot of a docker image file system](#creating-a-snapshot-of-a-docker-image-file-system)
    * [Output file name for a snapshot](#output-file-name-for-a-snapshot)
  * [Uploading a snapshot data to MinIO](#uploading-a-snapshot-data-to-minio)
  * [Create \& install snapshot CRD and k8s controller for it](#create--install-snapshot-crd-and-k8s-controller-for-it)
    * [Integration testing for the snapshot CRD controller](#integration-testing-for-the-snapshot-crd-controller)
  * [License](#license)

## Architecture

### Statechart diagram

![File location: docs/diagrams/integrityStatechartDiagram.png](/docs/diagrams/integrityStatechartDiagram.png?raw=true "Statechart diagram")

## Getting Started

### Clone repository and install dependencies

```
cd path/to/install
git clone https://github.com/ScienceSoft-Inc/integrity-sum.git
```

Download the named modules into the module cache

```
go mod download
```

## Demo-App

You can test this application in your CLI — Command Line Interface on local files and folders.
You can use it with option(flags) like:

1) **`-d`** (path to dir):

```
go run cmd/demo-app/main.go -d ./..
```

2) **`-a`** (hash algorithm):

```
go run cmd/demo-app/main.go -a sha256
go run cmd/demo-app/main.go -a SHA256
go run cmd/demo-app/main.go -a SHA256 -d ./..
```

3) **`-h`** (options docs):

```
go run cmd/demo-app/main.go -h
```

## :hammer: Installing components

### Running locally

The code only works running inside a pod in Kubernetes.
You need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster.
If you do not already have a cluster, you can create one by using `minikube`.
Example <https://minikube.sigs.k8s.io/docs/start/>

### Install Helm

Before using helm charts you need to install helm on your local machine.
You can find the necessary installation information at this link <https://helm.sh/docs/intro/install/>

### Configuration

To work properly, you first need to set the configuration files:

* environmental variables in the `.env` file
* values in the file `helm-charts/database-to-integrity-sum/values.yaml`
* values in the file `helm-charts/app-to-monitor/values.yaml`

## Quick start

### Using Makefile

You can use make function.
Runs all necessary cleaning targets and dependencies for the project:

```
make all
```

Remove an installed Helm deployments and stop minikube:

```
make stop
```

Building and running the project on a local machine:

```
make run
```

If you want to generate binaries for different platforms:

```
make compile
```

### Manual start

First of all, get a build image with `make buildtools`. It will be used to compile source code (Go & C/C++).

optional:

* SYSLOG_ENABLED=true, if syslog functionality is required;
* BEE2_ENABLED=true, if bee2 hash algorithm is required.

Build app: `make build`

Minikube start:

```
minikube start
```

Build docker image:

```
make docker
```

If you use `kind` instead of `minikube`, you may do `make kind-load-images` to preload image.

This command installs a chart archive.

```
helm install `release name` `path to a packaged chart`
```

There are some predefined targets in the Makefile for deployment:

* Install helm chart with app

    ```
    make helm-app
    ```

## Pay attention

If you want to use a hasher-sidecar, then you need to specify the following data in your deployment:
Pod annotations:

* `integrity-monitor.scnsoft.com/inject: "true"` - The sidecar injection annotation. If true, sidecar will be injected.
* `<monitoring process name>.integrity-monitor.scnsoft.com/monitoring-paths: etc/nginx,usr/bin` - This annotation introduces a process to be monitored and specifies its paths.
Annotation prefix should be the process name for instance `nginx`, `nginx.integrity-monitor.scnsoft.com/monitoring-paths`
Service account:
* `template:spec:serviceAccountName:` api-version-`hasher`
Share process namespace should be enabled.
* `template:shareProcessNamespace: true`

## Troubleshooting

Sometimes you may find that pod is injected with sidecar container as expected, check the following items:

1) The pod is in running state with `integrity` sidecar container injected and no error logs.
2) Check if the application pod has the correct annotations as described above.

___________________________

## :notebook_with_decorative_cover: Godoc extracts and generates documentation for Go programs

#### Presents the documentation as a web page

```go
godoc -http=:6060/integritySum
go doc packge.function_name
```

for example

```go
go doc pkg/api.Result
```

## :mag: Running tests

First of all you need to install mockgen:

```
go install github.com/golang/mock/mockgen@${VERSION_MOCKGEN}
```

Generate a mock:

```
go generate ./internal/core/ports/repository.go
go generate ./internal/core/ports/service.go
```

or

```
make generate
```

You need to go to the folder where the file is located *_test.go and run the following command:

```go
go test -v
```

for example

```go
cd ../pkg/api
go test -v
```

or

```
go test -v ./...
```

or

```
make test
```

## :mag: Running linter "golangci-lint"

```
golangci-lint run
```

## Including Bee2 library into the application

Use the Makefile target `bee2-lib` to build standalone static and shared binary for the bee2 library.
Use the env variable `BEE2_ENABLED=true` with `make build` to include bee2 library into the application. Then deployment may be updated with `--algorithm=BEE2` arg to select bee2 hashing algorithm.

Find more details about bee2 tools in the [Readme](internal/ffi/bee2/Readme.md).

## Enable MinIO

The code was tested with default `bitnami/minio` helm chart.

### Install standalone server

The following code will create the `minio` namespace and install a default MinIO server into it.

```bash
kubectl create ns minio
helm install minio --namespace=minio bitnami/minio
```

Refer to the original documentation [Bitnami Object Storage based on MinIO®](https://hub.docker.com/r/bitnami/minio/) for more details.

### Include into the project

To enable the MinIO in the project set the `minio.enabled` to `true` in the `helm-charts/app-to-monitor/values.yaml`.

## Syslog support

In order to enable syslog functionality following flags should be set:

* --syslog-enabled=true
* --syslog-host=syslog.host
* --syslog-port=syslog.port default 514
* --syslog-proto=syslog protocol default TCP

It could be done either through deployment or by integrity injector.

To enable syslog support in demo-app:

* Set `SYSLOG_ENABLED` environment variable to `true`
* Install demo app by running:

```
make helm-app
```

### Install syslog server

Syslog helm chart is included in demo app, to install it following steps should be done:

* Set `SYSLOG_ENABLED` environment variable to `true`
* Run: ```make helm-syslog```

### Syslog messages format

Syslog message format conforms to rfc3164 <https://www.ietf.org/rfc/rfc3164.txt>
e.g.

```
 <PRI> TIMESTAMP HOSTNAME TAG time=<event timestamp> event-type=<00001> service=<service name> namespace=<namespace name> cluster=<cluster name> message=<event message> file=<changed file name> reason=<event reason>\n`
```

* PRI - message priority, always 28 (LOG_WARNING | LOG_DAEMON)
* TIMESTAMP - time stamp format  "Jan _2 15:04:05"
* HOSTNAME - host/pod name
* TAG - process name with pid e.g. integrity-monitor[2]:

USER message consists from key=value pairs

* time=\<event timestamp\> , time stamp format  "Jan _2 15:04:05"
* event-type=\<00001\>
  * `00001` - "file content mismatch"
  * `00002` - "new file found"
  * `00003` - "file deleted"
  * `00004` - "heartbeat event"
* service=\<service name\>, monitoring service name e.g. `service=nginx`
* pod=app-nginx-integrity-579665544d-sh65t, monitoring pod name
* image=nginx:stable-alpine3.17, application image
* namespace=\<namespace name\>, pod namespace
* cluster=\<cluster name\>, service cluster name
* message=\<event message\>, e.g. `message=Restart deployment`
* file=\<changed file name\>, full file name with changes detected
* reason=\<event reason\>, restart reason e.g.:
  * `file content mismatch`
  * `new file found`
  * `file deleted`
  * `heartbeat event`

Message examples from syslog:

```log
Mar 31 10:46:19 app-nginx-integrity.default integrity-monitor[47]: time=Mar 31 10:46:19 event-type=0001 service=nginx pod=app-nginx-integrity-6bf9c6f4dd-xbvsp image=nginx:stable-alpine3.17 namespace=default cluster=local message=Restart pod app-nginx-integrity-6bf9c6f4dd-xbvsp file=etc/nginx/conf.d/default.conf reason=file content mismatch
Mar 31 11:02:26 app-nginx-integrity.default integrity-monitor[47]: time=Mar 31 11:02:26 event-type=0002 service=nginx pod=app-nginx-integrity-6bf9c6f4dd-rr7k6 image=nginx:stable-alpine3.17 namespace=default cluster=local message=Restart pod app-nginx-integrity-6bf9c6f4dd-rr7k6 file=etc/nginx/nfile reason=new file found
Mar 31 11:20:31 app-nginx-integrity.default integrity-monitor[69]: time=Mar 31 11:20:31 event-type=0003 service=nginx pod=app-nginx-integrity-6bf9c6f4dd-6t6rb image=nginx:stable-alpine3.17 namespace=default cluster=local message=Restart pod app-nginx-integrity-6bf9c6f4dd-6t6rb file=etc/nginx/nginx.conf reason=file deleted
Mar 31 11:25:10 app-nginx-integrity.default integrity-monitor[69]: time=Mar 31 11:25:10 event-type=0004 service=integrity-monitor pod=app-nginx-integrity-579665544d-sh65t image= namespace=default cluster=local message=health check file= reason=heartbeat event
```

Note: `<PRI>` - is not shown up in syslog server logs.

## Creating a snapshot of a docker image file system

You need to perform the following steps:

* export image file system to some local directory
* use snapshot command (`go run ./cmd/snapshot`) to create a snapshot of the exported early directories

It could be done either manually or by using predefined Makefile targets:

* `export-fs`

  Example of usage:

  ```bash
  IMAGE_EXPORT=integrity:latest make export-fs
  ```

* `snapshot`

  Example of usage:

  ```bash
  $ ALG=MD5 DIRS="app,bin" make snapshot
  ...
  created helm-charts/snapshot/files/snapshot.MD5
  f731846ea75e8bc9f76e7014b0518976  app/db/migrations/000001_init.down.sql
  96baa06f69fd446e1044cb4f7b28bc40  app/db/migrations/000001_init.up.sql
  353f69c28d8a547cbfa34c8b804501ba  app/integritySum
  ```

It is possible to combine the two commands into a single one:

```bash
IMAGE_EXPORT=integrity:latest DIRS="app,bin" make export-fs snapshot
```

In this case, the snapshot will be created with default (SHA256) algorithm and the snapshot will be stored as `helm-charts/snapshot/files/integrity:latest.sha256`.

### Output file name for a snapshot

The default location: `helm-charts/snapshot/files`

The default file name: `snapshot.<ALG>`, for example: `snapshot.MD5`

If you want to create a snapshot with a name that corresponds to an image, you should define the `IMAGE_EXPORT` variable for the `make snapshot` command. In this case, the output file will have the following format: `<default location>/<IMAGE_EXPORT>.<ALG>`.

Example: `helm-charts/snapshot/files/integrity:latest.sha256`.

## Uploading a snapshot data to MinIO

Reuired:

* generated with previous steps snapshot file(s)
* installed CRD for snapshot (see the [next](#create--install-snapshot-crd-and-k8s-controller-for-it) section)
* installed snapshot CRD controller (see the [next](#create--install-snapshot-crd-and-k8s-controller-for-it) section)

The snapshot data will be placed into the snapshot CRD and uploaded to the MinIO server with:

```bash
make helm-snapshot
```

With this command the `helm-charts/snapshot/files` dir will be scanned for the snapshot files which will be used to create the snapshot CRD(s).

```bash
$ ls -1 helm-charts/snapshot/files/
integrity:latest.md5
integrity:latest.sha256
integrity:latest.sha512
```

Then, generated CRDs will be created on the cluster. They might be manged by the `kubectl` command.

```bash
$ kubectl get snapshots
NAME                               IMAGE              UPLOADED   HASH
snapshot-integrity-latest-md5      integrity:latest   true       0c5f26feb07688edab0e71a2df04709a
snapshot-integrity-latest-sha256   integrity:latest   true       8a5f180beb77fa0575ecb43110df6d09
snapshot-integrity-latest-sha512   integrity:latest   true       3301c7429e05554013de17805e54a578
```

Finally, the snapshot controller will read the data from CRD(s) and upload it to the MinIO server.

With delete action on CRD the corresponding data from the MinIO server will be deleted as well.

## Create & install snapshot CRD and k8s controller for it

To create CRD related manifests and build the controller image for it use the following command:

```make
make crd-controller-build
```

To install the snapshot CRD and deploy it controller on the cluster use the following command:

```make
make crd-controller-deploy
```

Now we should be ready to manage snapshot CRDs on the cluster.

You may find more specific targets for the CRD & it controller in the  appropriate Makefile in the `snapshot-controller` directory.

### Integration testing for the snapshot CRD controller

Requirements:

* configured access to k8s cluster with installed and run MinIO service which will be used to store the snapshot data from the test CRD sample.
* placement of the MinIO service is hardcoded now to the `minio` namespace and `minio` service values. These values are used to find the MinIO credentials in the cluster and to perform port-forwarding to access the MinIO service and verify the data stored in it during the test.
* the snapshot controller should not be deployed on the cluster (we will test our code instead).

To perform integration test you may use the following Makefile target:

```bash
make crd-controller-test
```

It will update manifest for the snapshot CRD, install it to the cluster and then run the test. During the first run the `ginkgo` tool will be installed. This tool is used to run the integration test instead of default `go test`.

Notice:
Pay attention that this integration test performs on the current/real k8s cluster. The following command will show an information about the current cluster: `kubectl config current-context`.

## License

This project uses the MIT software license. See [full license file](https://github.com/ScienceSoft-Inc/integrity-sum/blob/main/LICENSE)
