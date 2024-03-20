PHONY: build run check format

export GOBIN ?= $(shell pwd)/.bin
GOFUMPT = $(GOBIN)/gofumpt

$(GOFUMPT): go.mod
	go install -v mvdan.cc/gofumpt

format: $(GOFUMPT)
	$(GOFUMPT) -w .

build:
	go build -o bin/main main.go

run:
	go run cmd/golangci-lint-temporalio/main.go

check:
	go run cmd/golangci-lint-temporalio/main.go -- ./test/example.go

