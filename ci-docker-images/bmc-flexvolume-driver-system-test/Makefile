REGISTRY ?= registry.oracledx.com/skeppare
IMAGE_NAME ?= bristol-bmc-flexvolume-driver-system-test
IMAGE ?= $(REGISTRY)/$(IMAGE_NAME)

VERSION ?= 1.0.0

.PHONY: build
build:
	docker build \
	    --build-arg=http_proxy \
	    --build-arg=https_proxy \
	    -t ${IMAGE_NAME}:${VERSION} \
	    -f Dockerfile \
	    .
	docker tag ${IMAGE_NAME}:${VERSION} ${IMAGE}:${VERSION}

.PHONY: push
push: build
	docker push ${IMAGE}:${VERSION}
