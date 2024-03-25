package main

import (
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/callables"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/serializable"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(callables.Analyzer, serializable.Analyzer)
}
