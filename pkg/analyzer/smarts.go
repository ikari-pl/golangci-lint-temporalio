package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func identifyCallable(pass *analysis.Pass) (workflows, activities Callables) {
	var knownActivities = make(map[ast.Expr]types.Type)
	var knownWorkflows = make(map[ast.Expr]types.Type)
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {

			if e, ok := n.(ast.Expr); ok {
				if debug > 50 {
					t := pass.TypesInfo.TypeOf(e)
					fmt.Printf("\t Type of %s is %s\n", e, t)
				}

				// for any RegisterActivity call on a go.temporal.io/sdk/worker.Worker type, record the function name
				// and the signature of the function being registered
				if callExpr, ok := n.(*ast.CallExpr); ok {
					if selector, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						xType := pass.TypesInfo.TypeOf(selector.X)
						if xType != nil && xType.String() == "go.temporal.io/sdk/worker.Worker" {
							if selector.Sel.Name == "RegisterActivity" {
								// record the function name and signature
								knownActivities[callExpr.Args[0]] = pass.TypesInfo.TypeOf(callExpr.Args[0])
							}
							if selector.Sel.Name == "RegisterWorkflow" {
								// record the function name and signature
								knownWorkflows[callExpr.Args[0]] = pass.TypesInfo.TypeOf(callExpr.Args[0])
							}
						}
					}
					if debug > 50 {
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
