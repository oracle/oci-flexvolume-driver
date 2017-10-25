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
DOCKER_IMAGE_TAG ?= 1.0.0

.PHONY: build
build:
	docker build --build-arg PROXY=${http_proxy} -t ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(DOCKER_REPO)
	docker push ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
