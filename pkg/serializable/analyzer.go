package serializable

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	goTypes "go/types"

	"github.com/ikari-pl/golangci-lint-temporalio/pkg/callables"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/external"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/internal/asttools"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/internal/types"
	"github.com/spf13/pflag"
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
	Analyzer.Flags.BoolVar(&debug, "debug-serializable", false,
		"Enable debug mode")
	Analyzer.Flags.BoolVar(&reportUnresolved, "report-unresolved", false,
		"Report unresolved workflow/activity names")
	Analyzer.Flags.BoolVar(&strictPointerMatch, "strict-pointer-match", false,
		"Require pointer types to match exactly, otherwise pointer vs underlying type is considered a match")
	pflag.CommandLine.AddGoFlagSet(&Analyzer.Flags)
}

var (
	debug              bool
	reportUnresolved   bool
	strictPointerMatch bool
)

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
		callee := c.Callee
		if callee == nil {
			caleePos := 2 // default to workflow identifier position in the call
			if c.Type == types.Activity {
				caleePos = 1
			}
			// we may not know the type, let's see if it's called by name
			// if it's called by name, we can look it up in the package
			// if it's not, we can't do anything
			bLit, isBasicLit := c.Expr.Args[caleePos].(*ast.BasicLit)
			if isBasicLit && bLit.Kind == token.STRING {
				// we have a string literal, we can look it up
				for _, o := range thisPkg.Workflows {
					if o.Name() == bLit.Value {
						callee = o
						break
					}
				}
			}
			if callee == nil && reportUnresolved {
				// can we make it a warning?
				pass.Reportf(c.Pos, "Could not resolve the type of the workflow/activity")
			}
		}
		for _, callArg := range c.CallArgs {
			actualT := pass.TypesInfo.TypeOf(callArg)
			if actualT != nil {
				// get argument name from callArg
				// if it's a struct, check if all fields are exported
				checkArgType(pass, c, actualT, callArg)
			}
		}

		// additionally, check if the type of the argument matches the argument type of the workflow/activity
		if callee != nil {
			signature := callee.Type().(*goTypes.Signature)
			checkArgumentCount(pass, c.Pos, callee.Name(), signature, c.CallArgs)
			checkArgumentTypes(pass, c.Pos, callee.Name(), signature, c.CallArgs)

			if debug {
				fmt.Printf("Call to %s at %s\n", c.Callee.Name(), pass.Fset.Position(c.Pos))
			}
		}
	}
	if debug {
		fmt.Printf("%d calls to workflows/activities checked\n", len(calls))
	}
	return nil, nil
}

func checkArgType(pass *analysis.Pass, c types.TemporalCall, actualT goTypes.Type, callArg ast.Expr) {
	var argName string
	ident, ok := callArg.(*ast.Ident)
	if ok {
		argName = ident.Name
	}
	if is, why := asttools.IsSerializable(actualT); !is {
		calleName := ""
		if c.Callee != nil {
			calleName = c.Callee.Name()
		}
		pass.Reportf(callArg.Pos(), "call argument `%s` (`%s`) is not serializable - it will not "+
			"be visible to `%s`, and will assume its zero value\n\treason: %s",
			argName, actualT.String(),
			calleName, why)
	}
}

func checkArgumentTypes(pass *analysis.Pass, pos token.Pos, callee string, signature *goTypes.Signature, callArgs []ast.Expr) {
	expectedParams := signature.Params().Len() - 1
	// we are going to check up to the maximum of expected and actual arguments
	argsCount := max(expectedParams, len(callArgs))
	// if the signature is variadic, we need to compare the count of arguments minus the variadic parameter
	if signature.Variadic() {
		argsCount = max(signature.Params().Len()-1, len(callArgs))
	}
	for argIdx := range argsCount {
		var expectedT, actualT goTypes.Type
		// Notes:
		// - expected args start with a context, that is not included in the call arguments
		// - for variadic functions, we need to compare the type of the variadic parameter ([]T)
		//   with the type of all trailing arguments (T)
		if signature.Variadic() && argIdx >= expectedParams-1 {
			expectedT = signature.Params().At(signature.Params().Len() - 1).Type().(*goTypes.Slice).Elem()
		} else {
			if argIdx < signature.Params().Len()-1 {
				expectedT = signature.Params().At(argIdx + 1).Type()
			}
		}
		if argIdx < len(callArgs) {
			actualT = pass.TypesInfo.TypeOf(callArgs[argIdx])
		}
		if expectedT == nil || actualT == nil {
			continue
		}
		if !goTypes.Identical(expectedT, actualT) {
			// is it a pointer vs non-pointer mismatch? (temporal handles these)
			if !strictPointerMatch {
				if ptr, ok := expectedT.(*goTypes.Pointer); ok {
					if goTypes.Identical(ptr.Elem(), actualT) {
						continue
					}
				}
				if ptr, ok := actualT.(*goTypes.Pointer); ok {
					if goTypes.Identical(ptr.Elem(), expectedT) {
						continue
					}
				}
				// and if expected type is a struct or a pointer to a struct, and actual is untyped nil, it's fine
				if _, ok := expectedT.Underlying().(*goTypes.Struct); ok {
					if actualT == goTypes.Typ[goTypes.UntypedNil] {
						continue
					}
				}
			}

			ordinal := numberToOrdinal(argIdx + 1)
			pass.Reportf(pos, "Type of %s argument to `%s` does not match the type of the workflow/activity\n"+
				"\tExpected: %s,\n\t     got: %s", ordinal, callee, expectedT, actualT)
		}
	}
}

func checkArgumentCount(pass *analysis.Pass,
	pos token.Pos,
	calleeName string,
	signature *goTypes.Signature,
	callArgs []ast.Expr,
) {
	expectedParams := signature.Params().Len() - 1
	if !signature.Variadic() {
		// for non variadic, we can check if the number of arguments is correct
		if expectedParams < len(callArgs) {
			pass.Reportf(pos, "Too many arguments to `%s` - expected %d, got %d", calleeName,
				expectedParams, len(callArgs))
		}
		if expectedParams > len(callArgs) {
			pass.Reportf(pos, "Too few arguments to `%s` - expected %d, got %d", calleeName,
				expectedParams, len(callArgs))
		}
	} else if len(callArgs) < signature.Params().Len()-1 {
		// for variadic, we can only check if the number of arguments is at least the number of non-variadic parameters
		pass.Reportf(pos, "Too few arguments to `%s` - expected at least %d, got %d", calleeName,
			signature.Params().Len()-1, len(callArgs))
	}
}

func identifyCalls(pass *analysis.Pass) []types.TemporalCall {
	var calls []types.TemporalCall
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
			if selector.Sel.Name != external.ExecuteWorkflow && selector.Sel.Name != external.ExecuteActivity {
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
			if xType != nil && xType.String() == external.ClientType {
				callee := call.Args[2]
				calleeID := asttools.IdentifierOf(callee)
				caleeObj := pass.TypesInfo.ObjectOf(calleeID)

				calls = append(calls, types.TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
					Expr:     call,
					Callee:   caleeObj,
					// skip the first two arguments (context, start options, and the callee)
					CallArgs: call.Args[3:],
					Type:     types.Workflow,
				})
				return true
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
			p, ok := o.(*goTypes.PkgName)
			if !ok {
				return true
			}
			// check if the package name is "go.temporal.io/sdk/workflow"
			if p.Imported().Path() == external.WorkflowPkg {
				callee := call.Args[1]
				var calleeID *ast.Ident
				var caleeObj goTypes.Object
				// if we expect the calee to be identified by a string literal
				if pass.TypesInfo.TypeOf(callee).String() == "string" {
					// if we have a string literal, we can look it up
					if lit, ok := callee.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						litValue := lit.Value
						for _, o := range pass.ResultOf[callables.Analyzer].(callables.Callables).Workflows {
							if o.Name() == litValue {
								caleeObj = o
								break
							}
						}
					}
					if caleeObj == nil {
						// the user may decide to report unresolved workflow/activity names
						// if their use-case should always point to package-local functions
						if reportUnresolved {
							// can we make it a warning?
							pass.Reportf(call.Pos(), "Could not resolve the type of the workflow/activity")
						}
						return true // not much we can do
					}
				}
				calleeID = asttools.IdentifierOf(callee)
				caleeObj = pass.TypesInfo.ObjectOf(calleeID)

				calls = append(calls, types.TemporalCall{
					Pos:      call.Pos(),
					FileName: pass.Fset.Position(call.Pos()).Filename,
					CallName: selector.Sel.Name,
					Expr:     call,
					Callee:   caleeObj,
					// skip the first two arguments (context, and the callee)
					CallArgs: call.Args[2:],
					Type:     types.Activity,
				})
			}

			return true
		})
	}
	return calls
}

func numberToOrdinal(n int) string {
	if n <= 0 {
		return "0"
	}
	if n%100 >= 11 && n%100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}
