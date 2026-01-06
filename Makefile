.PHONY: build clean test lint install run dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

build:
	go build $(LDFLAGS) -o auto ./cmd/auto

install:
	go install $(LDFLAGS) ./cmd/auto

clean:
	rm -f auto
	rm -rf dist/

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

run: build
	./auto

dev:
	go run $(LDFLAGS) ./cmd/auto

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

deps:
	go mod download
	go mod tidy

check: lint test

all: deps lint test build
