include ./make/*.mk

SHELL=/usr/bin/env bash -o pipefail
BUILD_TARGET=./

all: test
.PHONY: all

test: test/lint test/unit
.PHONY: test

test/lint:
	@shellcheck scripts/ocm/*.sh
.PHONY: lint

test/unit:
	go test -v ./cmd ./pkg/...
.PHONY: test

format:
	@echo "ToDo Implement me (format)!!"
.PHONY: format

build/cli:
	GOFLAGS=-mod=vendor go build -o=$(BUILD_TARGET) .
.PHONY: build

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: vendor/check
vendor/check: vendor/fix
	git diff --exit-code vendor/ go.*

.PHONY: vendor/fix
vendor/fix:
	go mod tidy
	go mod vendor
