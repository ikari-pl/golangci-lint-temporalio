package main

import (
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
