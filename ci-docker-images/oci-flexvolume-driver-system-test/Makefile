# Copyright 2017 The OCI Flexvolume Driver Authors
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

DOCKER_REPO ?= wcr.io
DOCKER_USER ?= oracle
DOCKER_IMAGE_NAME ?= oci-flexvolume-driver-system-test
VERSION ?= 1.0.0
IMAGE ?= ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}

.PHONY: build
build:
	docker build \
	    --build-arg http_proxy=${http_proxy} \
	    --build-arg https_proxy=${https_proxy} \
	    -t ${IMAGE}:${VERSION} .

.PHONY: push
push: build
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(DOCKER_REPO)
	docker push ${IMAGE}:${VERSION}
