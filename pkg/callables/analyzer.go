package callables

import (
	"flag"
	"fmt"
	"go/ast"
	goTypes "go/types"
	"os"
	"reflect"

	"golang.org/x/tools/go/analysis"

	"github.com/ikari-pl/golangci-lint-temporalio/pkg/external"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/internal/asttools"
	"github.com/ikari-pl/golangci-lint-temporalio/pkg/internal/types"
)

var Analyzer = &analysis.Analyzer{
	Name:       "TemporalIoCallables",
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

type Registration struct {
	Call            ast.CallExpr
	CalleeSignature *goTypes.Signature
	Type            types.TemporalIoCallType
}

func init() {
	tcFlags.BoolVar(&debug, "debug", false, "Enable debug mode")
}

// run is the main function of the analyzer,
// it identifies workflows and activities and exports them as Object Facts,
// as well as returning them as a result.
func run(pass *analysis.Pass) (interface{}, error) {
	workflows, activities, registrations := identify(pass)
	export(pass, workflows, activities)

	// now let's identify calls to these workflows and activities
	for _, r := range registrations {
		checkCalleeMatchesRegistration(pass, r)
	}
	return Callables{
		Workflows:  workflows,
		Activities: activities,
	}, nil
}

// export exports the identified workflows and activities as facts
// so that they can be used by other analyzers
// (note: sometimes we panic, in which case the export is not successful)
func export(pass *analysis.Pass, workflows []goTypes.Object, activities []goTypes.Object) {
	defer func() {
		// bug: we panic sometimes, let's recover
		if r := recover(); r != nil {
			fmt.Printf("WARN: Recovered from panic: %v\n", r)
		}
	}()
	for _, v := range workflows {
		// make sure v belongs to the package we're analyzing
		if v.Pkg() != pass.Pkg {
			continue
		}
		pass.ExportObjectFact(v, new(isActivity))
		if isDebug() {
			fmt.Printf("Workflow: %s\n", v)
		}
	}
	for _, v := range activities {
		if v.Pkg() != pass.Pkg {
			continue
		}
		pass.ExportObjectFact(v, new(isWorkflow))
		if isDebug() {
			fmt.Printf("Activity: %s\n", v)
		}
	}
}

func isDebug() bool {
	return debug
}

func identify(pass *analysis.Pass) (workflows, activities []goTypes.Object, registerCalls []Registration) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			var t types.TemporalIoCallType
			callExpr, methodName, isRegisterCall := asRegisterCall(n, pass)
			if !isRegisterCall {
				return true
			}
			switch methodName {
			case external.RegisterWorkflow:
				t = types.Workflow
				firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
				workflows = append(workflows, firstArgObj)
			case external.RegisterActivity:
				t = types.Activity
				firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
				activities = append(activities, firstArgObj)
			// you can also register an activity under the name of your choice with RegisterActivityWithOptions
			case external.RegisterActivityWithOptions:
				firstArgObj := pass.TypesInfo.ObjectOf(asttools.IdentifierOf(callExpr.Args[0]))
				optionsArgObj := asttools.IdentifierOf(callExpr.Args[1])
				// optionsArgObj is a struct of type worker.RegisterActivityOptions,
				// let's check if it has a Name field
				alt := identifyAsNamed(optionsArgObj, pass, callExpr, firstArgObj)
				if alt != nil {
					activities = append(activities, alt)
				}
				if firstArgObj == nil {
					// can be an inline function, but we don't support that
					position := pass.Fset.Position(callExpr.Pos())
					_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("WARN: First argument of RegisterActivityWithOptions is not a function reference: %s", position))
				} else {
					activities = append(activities, firstArgObj)
				}
			default:
				t = types.NotSupported
			}
			if isDebug() {
				t2 := pass.TypesInfo.TypeOf(callExpr.Fun)
				fmt.Printf("Type of %s is %s\n", callExpr.Fun, t2)
			}

			// for now, we only verify the signatures we can resolve (functions, not their names, passed as arguments)
			if t != types.NotSupported {
				sig, isSig := pass.TypesInfo.TypeOf(callExpr.Args[0]).(*goTypes.Signature)
				if isSig {
					registerCalls = append(registerCalls, Registration{
						Call:            *callExpr,
						CalleeSignature: sig,
						Type:            t,
					})
				}
			}
			return true
		})
	}
	return workflows, activities, registerCalls
}

func asRegisterCall(n ast.Node, pass *analysis.Pass) (*ast.CallExpr, string, bool) {
	e, ok := n.(ast.Expr)
	if !ok {
		return nil, "", false
	}
	if isDebug() {
		t := pass.TypesInfo.TypeOf(e)
		fmt.Printf("\t Type of %s is %s\n", e, t)
	}

	// for any RegisterActivity call on a go.temporal.io/sdk/worker.Worker type, record the function name
	// and the signature of the function being registered
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, "", false
	}
	selector, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, "", false
	}
	xType := pass.TypesInfo.TypeOf(selector.X)
	if !(xType != nil && xType.String() == external.WorkerType) {
		return nil, "", false
	}
	return callExpr, selector.Sel.Name, true
}

// identifyAsNamed checks if the options object has a Name field and returns an alternative object
// representing the activity with that name.
func identifyAsNamed(optionsArgObj *ast.Ident,
	pass *analysis.Pass,
	callExpr *ast.CallExpr,
	firstArgObj goTypes.Object,
) goTypes.Object {
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
	t, isStruct := optionsType.Type.Underlying().(*goTypes.Struct)
	if !isStruct {
		return nil
	}
	for i := range t.NumFields() {
		field := t.Field(i)
		if field.Name() == "Name" {
			sign, isSign := firstArgObj.Type().(*goTypes.Signature)
			if !isSign {
				return nil
			}
			altObj := goTypes.NewFunc(firstArgObj.Pos(), firstArgObj.Pkg(), field.String(), sign)
			fmt.Printf("Found an alternative name for activity: %s\n", altObj.Name())
			return altObj
		}
	}
	return nil
}

func checkCalleeMatchesRegistration(pass *analysis.Pass, registration Registration) {
	if registration.CalleeSignature == nil {
		// we can't resolve the callee's signature, so we can't check if it's a workflow or an activity
		return
	}

	if registration.CalleeSignature.Params().Len() < 1 {
		pass.Reportf(registration.Call.Pos(), "Workflow/activity must take at least one argument")
		return
	}

	switch registration.Type {
	case types.Workflow:
		// worfklow's first argument is always a workflow.Context
		if !external.WorkflowCtx.MatchString(registration.CalleeSignature.Params().At(0).Type().String()) {
			pass.Reportf(registration.Call.Pos(),
				"Workflow must take a workflow.Context as the first argument. Did you want to register an activity?")
		}
	case types.Activity:
		// activity's first argument is always a context.Context
		argType := registration.CalleeSignature.Params().At(0).Type().String()
		if argType != external.ActivityCtx {
			msg := "Activity must take a context.Context as the first argument"
			if !external.WorkflowCtx.MatchString(argType) {
				msg += ". Did you want to register a workflow?"
			}
			pass.Reportf(registration.Call.Pos(), msg, nil)
		}
	case types.NotSupported:
	// pass
	default:
		panic("unsupported Temporal.io call type")
	}
}
