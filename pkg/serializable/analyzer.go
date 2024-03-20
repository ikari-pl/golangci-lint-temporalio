package serializable

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/ikari-pl/golangci-lint-temporalio/pkg/asttools"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/callables"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/externalDeps"
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
	Analyzer.Flags.BoolVar(&debug, "debug-serializable", false, "Enable debug mode")
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

	// get all places where we call a workflow or an activity
	calls := identifyCalls(pass)
	for _, c := range calls {
		fmt.Println(c.Callee)
	}

	// Check that all arguments and return values of detected Temporal.io
	// workflows and activities contain serializable fields only.

	// Report any violations found

	return nil, nil
}

// TemporalCall represents a detected Temporal.io workflow or activity invocation.
type TemporalCall struct {
	Pos      token.Pos
	FileName string
	CallName string
	Expr     *ast.CallExpr
	Callee   types.Object
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
			// and for activities, it's: workflow.ExecuteActivity(ctx, HelloWorldActivity, name)
			// where workflow is import "go.temporal.io/sdk/workflow"

			xType := pass.TypesInfo.TypeOf(x)
			if xType != nil && xType.String() == externalDeps.ClientType {
				calls = append(calls, TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
				})
			}

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

			if len(call.Args) < 2 {
				// warn: not enough arguments, should never happen
			}
			callee := call.Args[1]
			calleeId := asttools.IdentifierOf(callee)
			caleeObj := pass.TypesInfo.ObjectOf(calleeId)

			// check if the package name is "workflow" and the import path is "go.temporal.io/sdk/workflow"
			if p.Imported().Path() == externalDeps.WorkflowPkg {
				calls = append(calls, TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
					Expr:     call,
					Callee:   caleeObj,
				})
			}

			return true
		})
	}
	return calls
}
