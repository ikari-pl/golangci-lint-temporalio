package serializable

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"unicode"

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
		callables.Analyzer,
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
	thisPkg := pass.ResultOf[callables.Analyzer].(callables.Callables)
	if debug {
		fmt.Printf("Found %d workflows and %d activities\n", len(thisPkg.Workflows), len(thisPkg.Activities))
	}

	// get all places where we call a workflow or an activity
	calls := identifyCalls(pass)
	for _, c := range calls {
		flowArgs := c.Expr.Args[2:]
		for _, arg := range flowArgs {
			t := pass.TypesInfo.TypeOf(arg)
			// if t is a struct, or a pointer to a struct, check if it's serializable
			if t != nil {
				if s, ok := t.Underlying().(*types.Struct); ok {
					for i := 0; i < s.NumFields(); i++ {
						f := s.Field(i)
						if len(f.Name()) > 0 && unicode.IsLower(rune(f.Name()[0])) {
							pass.Reportf(c.Pos, "Field `%s` of `%s` is not exported - it will not "+
									"be visible on the receiving end, and will assume its zero value", f.Name(), c.Callee.Name())
						}
						if !asttools.IsSerializable(f.Type()) {
							pass.Reportf(c.Pos, "Field `%s` of `%s` is not serializable - it will not "+
									"be visible on the receiving end, and will assume its zero value", f.Name(), c.Callee.Name())
						}
					}
				}
			}
		}
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
			if selector.Sel.Name != "ExecuteWorkflow" && selector.Sel.Name != "ExecuteActivity" {
				return true
			}
			x, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			// for workflows, we find:   client.ExecuteWorkflow(ctx, StartWorkflowOptions{}, "World")
			// and for activities, it's: workflow.ExecuteActivity(ctx, HelloWorldActivity, name)
			// where workflow is import "go.temporal.io/sdk/workflow"
			if len(call.Args) < 2 {
				// warn: not enough arguments, should never happen
				panic("not enough arguments to be a valid ExecuteWorkflow/Activity call")
			}
			xType := pass.TypesInfo.TypeOf(x)
			if xType != nil && xType.String() == externalDeps.ClientType {
				callee := call.Args[2]
				calleeId := asttools.IdentifierOf(callee)
				caleeObj := pass.TypesInfo.ObjectOf(calleeId)

				calls = append(calls, TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
					Expr:     call,
					Callee:   caleeObj,
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
			// check if the package name is "go.temporal.io/sdk/workflow"
			if p.Imported().Path() == externalDeps.WorkflowPkg {
				callee := call.Args[1]
				calleeId := asttools.IdentifierOf(callee)
				caleeObj := pass.TypesInfo.ObjectOf(calleeId)

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
