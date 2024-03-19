package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "temporalio",
	Doc:  "Checks that all temporal.io arguments and return values contain serializable fields only.",
	Run:  run,
}

var debug int = 1

// TemporalCall represents a detected Temporal.io workflow or activity invocation.
type TemporalCall struct {
	Pos      token.Pos
	FileName string
	CallName string
}

type Callables = map[ast.Expr]types.Type

func run(pass *analysis.Pass) (interface{}, error) {
	workflows, activities := identifyCallable(pass)
	if debug > 0 {
		for k, v := range workflows {
			pass.Reportf(k.Pos(), "Workflow: %s", v)
		}
		for k, v := range activities {
			pass.Reportf(k.Pos(), "Activity: %s", v)
		}
	}
	calls := identifyCalls(pass)
	for _, call := range calls {
		pass.Reportf(call.Pos, "Temporal call to %s in %s", call.CallName, call.FileName)
	}

	return nil, nil
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
			if xType != nil && xType.String() == "go.temporal.io/sdk/client.Client" {
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
			if p.Imported().Path() == "go.temporal.io/sdk/workflow" {
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
