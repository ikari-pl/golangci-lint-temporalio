PHONY: build run check format

export GOBIN ?= $(shell pwd)/.bin
GOFUMPT = $(GOBIN)/gofumpt

$(GOFUMPT): go.mod
	go install -v mvdan.cc/gofumpt

format: $(GOFUMPT)
	$(GOFUMPT) -w .

build:
	go build

demo:
	go run main.go -- ./test/example.go
