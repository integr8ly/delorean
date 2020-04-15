include ./make/*.mk

SHELL=/usr/bin/env bash -o pipefail
BUILD_TARGET=./

all: test
.PHONY: all

test: test/lint test/unit
.PHONY: test

test/lint:
	@echo "ToDo Implement me (test/lint)!!"
.PHONY: lint

test/unit:
	go test -v ./cmd
.PHONY: test

format:
	@echo "ToDo Implement me (format)!!"
.PHONY: format

build/cli:
	go build -o=$(BUILD_TARGET) .
.PHONY: build

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: vendor/check
vendor/check: vendor/fix
	git diff --exit-code vendor/

.PHONY: vendor/fix
vendor/fix:
	go mod tidy
	go mod vendor
