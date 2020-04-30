
CONTAINER_ENGINE ?= docker

DELOREAN_IMAGE=quay.io/integreatly/delorean-cli:latest

.PHONY: image/build
image/build: export BUILD_TARGET=./build
image/build: build/cli
	@${CONTAINER_ENGINE} build -t ${DELOREAN_IMAGE} -f build/Dockerfile .

image/test: export TEST_DELOREAN_IMAGE=${DELOREAN_IMAGE}
image/test: export TEST_CONTAINER_ENGINE=${CONTAINER_ENGINE}
.PHONY: image/test
image/test: image/build
	@./scripts/image/test
