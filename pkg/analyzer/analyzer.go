package analyzer

import (
	"flag"
	"fmt"

	"github.com/ikari-pl/golangci-lint-temporalio/pkg/callables"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "TemporalioSerializableFields",
	Doc:  "Checks that all temporal.io arguments and return values contain serializable fields only.",
	Run:  run,
	Requires: []*analysis.Analyzer{
		callables.TemporalCallables,
	},
	Flags: flag.FlagSet{},
}

func init() {
	Analyzer.Flags.BoolVar(&debug, "debug", false, "Enable debug mode")
}

var debug bool

func run(pass *analysis.Pass) (interface{}, error) {
	if debug {
		pass.Reportf(pass.Files[0].Pos(), "Debug mode is on")
	}

	// Import facts about detected Temporal.io workflows and activities
	// from the callables analyzer
	thisPkg := pass.ResultOf[callables.TemporalCallables].(callables.Callables)
	fmt.Print(thisPkg)

	// Import facts about detected Temporal.io workflows and activities
	// from the callables analyzer
	fmt.Print(pass.AllObjectFacts())

	// Check that all arguments and return values of detected Temporal.io
	// workflows and activities contain serializable fields only.

	// Report any violations found

	return nil, nil
}
