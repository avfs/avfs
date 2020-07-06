##
##	Copyright 2020 The AVFS authors
##
##	Licensed under the Apache License, Version 2.0 (the "License");
##	you may not use this file except in compliance with the License.
##	You may obtain a copy of the License at
##
##		http://www.apache.org/licenses/LICENSE-2.0
##
##	Unless required by applicable law or agreed to in writing, software
##	distributed under the License is distributed on an "AS IS" BASIS,
##	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
##	See the License for the specific language governing permissions and
##	limitations under the License.
##

## See https://tech.davis-hansson.com/p/make/
SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

## Local variables
GO:=GO111MODULE=on go
GOPATH:=$(shell go env GOPATH)
DOCKER_IMAGE:=avfs-docker
COVERAGE_FILE:=coverage.txt
RUNTEST?=.
COUNT?=5

.PHONY: all
all: golangci dockertest

.PHONY: build
build:
	@$(GO) build ./...

.PHONY: env
env:
	@$(GO) version && echo "PATH=$(PATH)" && $(GO) env

.PHONY: fmt_install
fmt_install:
	@if [ -z $(shell which gofumpt) ]; then
		$(GO) get mvdan.cc/gofumpt
	fi

.PHONY: fmt
fmt: fmt_install
	@gofumpt -l -s -w -extra .

.PHONY: vet
vet:
	@$(GO) vet -all ./...

.PHONY:golangci_install
golangci_install:
	@if [ -z $(shell which golangci-lint) ]; then
		## get the latest tagged version of golangci-lint
		version=`git ls-remote --tags --refs --sort="v:refname" https://github.com/golangci/golangci-lint/ | tail -n1 | sed "s/.*\///"`

		## binary will be $(shell go env GOPATH)/bin/golangci-lint
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $$version
	fi

.PHONY:golangci
golangci: golangci_install
	@$(GOPATH)/bin/golangci-lint run

.PHONY: coverage_init
coverage_init:
	@install -m 777 /dev/null $(COVERAGE_FILE)

.PHONY: test
test: env coverage_init
	@$(GO) test -run=$(RUNTEST) -race -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...

.PHONY: cover
cover:
	@$(GO) tool cover -html=$(COVERAGE_FILE)

.PHONY: race
race:
	@$(GO) test -tags=datarace -run=TestRace -race -v -count=$(COUNT) ./...

.PHONY: bench
bench:
	@$(GO) test -run=^a -bench=. -benchmem -count=5 ./...

.PHONY: dockerbuild
dockerbuild:
	@docker build . -t $(DOCKER_IMAGE)

.PHONY: dockertest
dockertest: dockerbuild coverage_init
	-@docker run -ti $(DOCKER_IMAGE)
	-@docker cp `docker ps -alq`:/go/src/$(COVERAGE_FILE) $(COVERAGE_FILE)

.PHONY: dockerconsole
dockerconsole: dockerbuild
	@docker run -ti $(DOCKER_IMAGE) /bin/bash

.PHONY: dockerprune
dockerprune:
	@docker system prune -f
