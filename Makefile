APP=otel-example
GOPATH=$(shell env | grep GOPATH | cut -d'=' -f 2)

SOURCE=./...
REVISION=$(shell git rev-list -1 HEAD)
TAG=$(shell git tag -l --points-at HEAD | tail -1)
ifeq ($(TAG),)
TAG=$(REVISION)
endif
BR=$(shell git rev-parse --abbrev-ref HEAD)
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

export GOPRIVATE=gitlab.com*


build:
	go install -mod=mod -v -ldflags "-s -X $(PKGPATH).appName=$(APP) -X $(PKGPATH).gitCommit=$(REVISION) -X $(PKGPATH).gitBranch=$(BR) -X $(PKGPATH).appVersion=$(TAG) -X $(PKGPATH).buildDate=$(DATE)" $(SOURCE)

run:
	go run -mod=mod -v -ldflags "-s -X $(PKGPATH).appName=$(APP) -X $(PKGPATH).gitCommit=$(REVISION) -X $(PKGPATH).gitBranch=$(BR) -X $(PKGPATH).appVersion=$(TAG) -X $(PKGPATH).buildDate=$(DATE)" $(SOURCE) server

modvendor:
	- rm go.sum
	go build -mod=mod -v $(SOURCE)
	go mod tidy
	go mod vendor

jaeger:
	docker run --rm --name jaeger -e JAEGER_DISABLED=true --network bridge -p 16686:16686 \
	-p 14250:14250 \
	-p 14268:14268 \
	-p 14269:14269 \
	-p 9411:9411 \
	jaegertracing/all-in-one:1.31
