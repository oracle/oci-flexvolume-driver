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

BUILD := $(shell git describe --always --dirty)
# Allow overriding for release versions
# Else just equal the build (git hash)
VERSION ?= ${BUILD}
GOOS ?= linux
GOARCH ?= amd64
REGISTRY ?= wcr.io
DOCKER_REGISTRY_USERNAME ?= oracle
TEST_IMAGE ?= $(REGISTRY)/$(DOCKER_REGISTRY_USERNAME)/oci-flexvolume-driver-test

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
	    -ldflags="-s -w -X main.version=${VERSION} -X main.build=${BUILD}" \
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

.PHONY: build-test-image
build-test-image:
	docker build -t ${TEST_IMAGE}:${VERSION} -f Dockerfile.test .

.PHONY: push-test-image
push-test-image: build-test-image
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	docker push ${TEST_IMAGE}:${VERSION}

.PHONY: system-test-config
system-test-config:
ifndef OCI_API_KEY
ifndef OCI_API_KEY_VAR
    $(error "OCI_API_KEY or OCI_API_KEY_VAR must be defined")
else
    $(eval OCI_API_KEY:=/tmp/oci_api_key.pem)
    $(eval export OCI_API_KEY)
    $(shell echo "$${OCI_API_KEY_VAR}" | openssl enc -base64 -d -A > $(OCI_API_KEY))
endif
endif
ifndef INSTANCE_KEY
ifndef INSTANCE_KEY_VAR
    $(error "INSTANCE_KEY or INSTANCE_KEY_VAR must be defined")
else
    $(eval INSTANCE_KEY:=/tmp/instance_key)
    $(eval export INSTANCE_KEY)
    $(shell echo "$${INSTANCE_KEY_VAR}" | openssl enc -base64 -d -A > $(INSTANCE_KEY))
    $(shell chmod 600 $(INSTANCE_KEY))
endif
endif

.PHONY: system-test
system-test: system-test-config
	docker run -it \
        -e OCI_API_KEY=$(OCI_API_KEY) \
        -v $(OCI_API_KEY):$(OCI_API_KEY) \
        -e INSTANCE_KEY=$(INSTANCE_KEY) \
        -v $(INSTANCE_KEY):$(INSTANCE_KEY) \
        -e MASTER_IP=$$MASTER_IP \
        -e SLAVE0_IP=$$SLAVE0_IP \
        -e SLAVE1_IP=$$SLAVE1_IP \
        -e WERCKER_API_TOKEN=$$WERCKER_API_TOKEN \
        -e HTTPS_PROXY=$$HTTPS_PROXY \
        ${TEST_IMAGE}:${VERSION} ${TEST_IMAGE_ARGS}

.PHONY: version
version:
	@echo ${VERSION}
