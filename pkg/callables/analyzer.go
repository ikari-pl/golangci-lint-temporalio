package callables

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"reflect"

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

const (
	WorkerType  = "go.temporal.io/sdk/worker.Worker"
	ClientType  = "go.temporal.io/sdk/client.Client"
	WorkflowPkg = "go.temporal.io/sdk/workflow"
)

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
		if isDebug() {
			fmt.Printf("Activity: %s\n", v)
		}
	}
	calls := identifyCalls(pass)
	for _, call := range calls {
		pass.Reportf(call.Pos, "Temporal call to %s in %s", call.CallName, call.FileName)
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
						if xType != nil && xType.String() == WorkerType {
							if selector.Sel.Name == "RegisterActivity" {
								firstArgObj := pass.TypesInfo.ObjectOf(identifierOf(callExpr.Args[0]))
								knownActivities = append(knownActivities, firstArgObj)
							}
							if selector.Sel.Name == "RegisterWorkflow" {
								firstArgObj := pass.TypesInfo.ObjectOf(identifierOf(callExpr.Args[0]))
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

// identifierOf returns the *ast.Ident for the given *ast.Expr, whatever it is.
func identifierOf(e ast.Expr) *ast.Ident {
	switch e := e.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return identifierOf(e.Sel)
	case *ast.StarExpr:
		return identifierOf(e.X)
	case *ast.CallExpr:
		return identifierOf(e.Fun)
	// pointer to a function
	case *ast.FuncType:
		return identifierOf(e)
	// pointer to a struct (represented as unary AND of a composite literal)
	case *ast.UnaryExpr:
		return identifierOf(e.X)
	// continuation of pointer to a struct (represented as composite literal)
	case *ast.CompositeLit:
		return identifierOf(e.Type)
	default:
		return nil
	}
}

func identifyCalls(pass *analysis.Pass) []TemporalCall {
	var calls []TemporalCall
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if !(selector.Sel.Name == "ExecuteWorkflow" || selector.Sel.Name == "ExecuteActivity") {
				return true
			}
			x, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			// client.ExecuteWorkflow(ctx, StartWorkflowOptions{}, "World")
			xType := pass.TypesInfo.TypeOf(x)
			if xType != nil && xType.String() == ClientType {
				calls = append(calls, TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
				})
			}
			// and for activities, it's: workflow.ExecuteActivity(ctx, HelloWorldActivity, name)
			// where workflow is import "go.temporal.io/sdk/workflow"

			// we need to resolve x.Name going up the scope chain to find the package name
			// and then check if it's a package we care about

			// get the scope at call.Pos()
			// get the package name from the scope
			// check if the package name is "workflow" and the import path is "go.temporal.io/sdk/workflow"
			// if so, record the call
			// if not, continue
			// if we reach the top of the scope chain, continue
			// if we reach the top of the file, continue

			// get the scope at call.Pos()
			scope := pass.TypesInfo.ObjectOf(x).Parent()
			if scope == nil {
				return true
			}
			o := scope.Lookup(x.Name)
			if o == nil {
				return true
			}
			// get the package name from the scope if o is a *types.PkgName
			p, ok := o.(*types.PkgName)
			if !ok {
				return true
			}

			// check if the package name is "workflow" and the import path is "go.temporal.io/sdk/workflow"
			if p.Imported().Path() == WorkflowPkg {
				calls = append(calls, TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
				})
			}

			return true
		})
	}
	return calls
}
