# Include go binaries into path
export PATH := $(GOPATH)/bin:$(PATH)

BUILD=$(shell date +%FT%T)
VERSION= $(shell git rev-parse --short HEAD)
CURRENT_BRANCH_NAME= $(shell git rev-parse --abbrev-ref HEAD)
LDFLAGS=-ldflags "-w -s -X main.Version=${VERSION} -X main.Build=${BUILD}"

CURDIR := $(shell pwd)
GOBIN := $(CURDIR)/bin/
ENV:=GOBIN=$(GOBIN)

SOURCE_PATH := GOBIN=$(GOBIN) CURDIR=$(CURDIR) TEST_SOURCE_PATH=$(PWD) CURRENT_BRANCH_NAME=$(CURRENT_BRANCH_NAME)

# teamcity
install: mod  ## Run installing
	@echo "Environment installed"

test: ## Run tests
	rm -f coverage.out coverage.html
	go test -race -cover -coverprofile=$(PWD)/coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	rm coverage.out


test-top: ## Run tests
	rm -f coverage.out coverage.html
	go test -cover -coverprofile=$(PWD)/coverage.out ./
	go tool cover -html=coverage.out -o coverage.html
	rm coverage.out

test-text: ## Run text tests
	rm -f coverage.text.out coverage.text.html
	go test -cover -coverprofile=$(PWD)/coverage.text.out ./internal/text
	go tool cover -html=coverage.text.out -o coverage.text.html
	rm coverage.text.out

# full cleaning. Don't use it: it removes outside golang packages for all projects
clean: ## Remove build artifacts
	@echo "======================================================================"
	@echo "Run clean"
	go clean -i -r -x -cache -testcache -modcache

clean-cache: ## Clean golang cache
	@echo "clean-cache started..."
	go clean -cache
	go clean -testcache
	@echo "clean-cache complete!"

clean-vendor: ## Remove vendor folder
	@echo "clean-vendor started..."
	rm -fr ./vendor
	@echo "clean-vendor complete!"

clean-all: clean clean-vendor clean-cache

mod-action-%:
	@echo "Run go mod ${*}...."
	GO111MODULE=on go mod $*
	@echo "Done go mod  ${*}"

mod: mod-action-verify mod-action-tidy mod-action-vendor mod-action-download mod-action-verify ## Download all dependencies

tests-filters: ## Testing filters
	@echo "======================================================================"
	@echo "Run tests-filters"
	@for dir in ./clutch-api-server/elastic/indexes/listpageproviders/filters/*; \
	do (cd $$dir && TEST_TIMEOUT=0 $(SOURCE_PATH) go test -test.timeout 0 --check.v --check.vv --check.format=teamcity ./...) || exit $$?; done


help: ## Show this help
	@egrep -h '\s##\s' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

deps:	## Install dependencies for development
	@echo "======================================================================"
	@echo 'MAKE: deps...'
	mkdir -p $(GOBIN)
	$(ENV) go install github.com/golang/mock/mockgen@v1.6.0

mock:
	@mkdir -p ./internal/cache/spm/mmock
	@rm -f ./internal/cache/spm/mmock/*.go
	#./bin/mockgen -package mmock -source=./spmcache/cache.go ISPMAPICache > ./spmcache/mmock/remote_spm_cache_mock.go
	./bin/mockgen -package mmock -source=./internal/cache/spm/cache.go > ./internal/cache/spm/mmock/spm_cache_mock.go

example3:
	cd ./examples/example_3 && go run ./main.go