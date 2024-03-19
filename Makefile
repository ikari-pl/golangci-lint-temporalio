PHONY: build run

build:
	go build -o bin/main main.go

run:
	go run cmd/golangci-lint-temporalio/main.go

check:
	go run cmd/golangci-lint-temporalio/main.go -- ./test/example.go