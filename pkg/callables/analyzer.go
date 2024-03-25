package callables

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"reflect"

	"github.com/ikari-pl/golangci-lint-temporalio/pkg/asttools"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/external"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:       "TemporalioCallables",
	Doc:        "Detects registrations of, and calls to Temporal.io workflows and activities",
	Run:        run,
	Flags:      tcFlags,
	FactTypes:  []analysis.Fact{new(isWorkflow), new(isActivity), new(isWorkflowCall), new(isActivityCall)},
	ResultType: reflect.TypeOf(Callables{}),
}

var (
	tcFlags flag.FlagSet
	debug   bool
)

func init() {
	tcFlags.BoolVar(&debug, "debug", false, "Enable debug mode")
}

func run(pass *analysis.Pass) (interface{}, error) {
	workflows, activities := identify(pass)
	/*
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
	*/
	return Callables{
		Workflows:  workflows,
		Activities: activities,
	}, nil
}

func isDebug() bool {
	return debug
}

func identify(pass *analysis.Pass) (workflows, activities []types.Object) {
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
						if xType != nil && xType.String() == external.WorkerType {
							if selector.Sel.Name == external.RegisterActivity {
								firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
								knownActivities = append(knownActivities, firstArgObj)
							}
							if selector.Sel.Name == external.RegisterWorkflow {
								firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
								knownWorkflows = append(knownWorkflows, firstArgObj)
							}
							// you can also register an activity under the name of your choice with RegisterActivityWithOptions
							if selector.Sel.Name == external.RegisterActivityWithOptions {
								firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
								optionsArgObj := asttools.IdentifierOf(callExpr.Args[1])
								// optionsArgObj is a struct of type worker.RegisterActivityOptions,
								// let's check if it has a Name field
								alt := identifyAsNamed(optionsArgObj, pass, callExpr, firstArgObj)
								if alt != nil {
									knownActivities = append(knownActivities, alt)
								}

								knownActivities = append(knownActivities, firstArgObj)
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

// identifyAsNamed checks if the options object has a Name field and returns an alternative object
// representing the activity with that name.
func identifyAsNamed(optionsArgObj *ast.Ident,
	pass *analysis.Pass,
	callExpr *ast.CallExpr,
	firstArgObj types.Object,
) types.Object {
	if optionsArgObj == nil {
		return nil
	}
	optionsType, ok := pass.TypesInfo.Types[callExpr.Args[1]]
	if !ok {
		return nil
	}
	if optionsType.Type.String() != "go.temporal.io/sdk/worker.RegisterActivityOptions" {
		return nil
	}
	t, isStruct := optionsType.Type.Underlying().(*types.Struct)
	if !isStruct {
		return nil
	}
	for i := range t.NumFields() {
		field := t.Field(i)
		if field.Name() == "Name" {
			sign, isSign := firstArgObj.Type().(*types.Signature)
			if !isSign {
				return nil
			}
			altObj := types.NewFunc(firstArgObj.Pos(), firstArgObj.Pkg(), field.String(), sign)
			fmt.Printf("Found an alternative name for activity: %s\n", altObj.Name())
			return altObj
		}
	}
	return nil
}
