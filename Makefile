# Copyright 2017 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BIN := oci
BUILD_DIR := dist
BIN_DIR := ${BUILD_DIR}/bin
VERSION := $(shell git describe --always --dirty)

GOOS ?= linux
GOARCH ?= amd64

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

.PHONY: all
all: clean test build build-integration-tests

.PHONY: gofmt
gofmt:
	@./hack/check-gofmt.sh ${SRC_DIRS}

.PHONY: golint
golint:
	@./hack/check-golint.sh ${SRC_DIRS}

.PHONY: govet
govet:
	@./hack/check-govet.sh ${SRC_DIRS}

.PHONY: test
test:
	@./hack/test.sh $(SRC_DIRS)

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

.PHONY: build
build:
	mkdir -p ${BIN_DIR}
	GOOS=${GOOS} \
	    CGO_ENABLED=0 \
	    GOARCH=${GOARCH} \
	    go build \
	    -i \
	    -v \
	    -ldflags="-s -w -X main.version=${VERSION}" \
	    -o ${BIN_DIR}/${BIN} ./cmd/oci/

.PHONY: build-integration-tests
build-integration-tests:
	mkdir -p ${BIN_DIR}
	GOOS=${GOOS} \
	    CGO_ENABLED=0 \
	    GOARCH=${GOARCH} \
	    go test \
	    -v \
	    -c \
	    -i \
	    -o ${BIN_DIR}/integration-tests \
	    ./test/integration
