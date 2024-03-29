package external

const (
	WorkerType  = "go.temporal.io/sdk/worker.Worker"
	ClientType  = "go.temporal.io/sdk/client.Client"
	WorkflowPkg = "go.temporal.io/sdk/workflow"

	WorkflowCtx = "go.temporal.io/sdk/internal.Context"
	ActivityCtx = "context.Context"

	RegisterActivity            = "RegisterActivity"
	RegisterActivityWithOptions = "RegisterActivityWithOptions"
	RegisterWorkflow            = "RegisterWorkflow"

	ExecuteActivity = "ExecuteActivity"
	ExecuteWorkflow = "ExecuteWorkflow"
)
