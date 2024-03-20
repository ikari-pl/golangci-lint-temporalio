package callables

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"reflect"

	"github.com/ikari-pl/golangci-lint-temporalio/pkg/asttools"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/externalDeps"
	"golang.org/x/tools/go/analysis"
)

var TemporalCallables = &analysis.Analyzer{
	Name:       "TemporalioCallables",
	Doc:        "Detects registrations of, and calls to Temporal.io workflows and activities",
	Run:        run,
	Flags:      tcFlags,
	FactTypes:  []analysis.Fact{new(isWorkflow), new(isActivity), new(isWorkflowCall), new(isActivityCall)},
	ResultType: reflect.TypeOf(Callables{}),
}

var tcFlags flag.FlagSet
var debug bool

func init() {
	tcFlags.BoolVar(&debug, "debug", false, "Enable debug mode")
}

func run(pass *analysis.Pass) (interface{}, error) {
	workflows, activities := identifyCallable(pass)
	for _, v := range workflows {
		pass.ExportObjectFact(v, new(isActivity))
		if isDebug() {
			fmt.Printf("Workflow: %s\n", v)
		}
	}
	for _, v := range activities {
		pass.ExportObjectFact(v, new(isWorkflow))
		if isDebug() {
			fmt.Printf("Activity: %s\n", v)
		}
	}
	return Callables{
		Workflows:  workflows,
		Activities: activities,
	}, nil
}

func isDebug() bool {
	return debug
}

func identifyCallable(pass *analysis.Pass) (workflows, activities []types.Object) {
	var knownActivities []types.Object
	var knownWorkflows []types.Object
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if e, ok := n.(ast.Expr); ok {
				if isDebug() {
					t := pass.TypesInfo.TypeOf(e)
					fmt.Printf("\t Type of %s is %s\n", e, t)
				}

				// for any RegisterActivity call on a go.temporal.io/sdk/worker.Worker type, record the function name
				// and the signature of the function being registered
				if callExpr, ok := n.(*ast.CallExpr); ok {
					if selector, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						xType := pass.TypesInfo.TypeOf(selector.X)
						if xType != nil && xType.String() == externalDeps.WorkerType {
							if selector.Sel.Name == "RegisterActivity" {
								firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
								knownActivities = append(knownActivities, firstArgObj)
							}
							if selector.Sel.Name == "RegisterWorkflow" {
								firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
								knownWorkflows = append(knownWorkflows, firstArgObj)
							}
						}
					}
					if isDebug() {
						t2 := pass.TypesInfo.TypeOf(callExpr.Fun)
						fmt.Printf("Type of %s is %s\n", callExpr.Fun, t2)
					}
				}
			}
			return true
		})
	}
	return knownWorkflows, knownActivities
}
