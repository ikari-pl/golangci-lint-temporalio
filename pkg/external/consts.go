package external

const (
	WorkerType  = "go.temporal.io/sdk/worker.Worker"
	ClientType  = "go.temporal.io/sdk/client.Client"
	WorkflowPkg = "go.temporal.io/sdk/workflow"

	RegisterActivity            = "RegisterActivity"
	RegisterActivityWithOptions = "RegisterActivityWithOptions"
	RegisterWorkflow            = "RegisterWorkflow"

	ExecuteActivity = "ExecuteActivity"
	ExecuteWorkflow = "ExecuteWorkflow"
)
