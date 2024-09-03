package external

import "regexp"

const (
	WorkerType  = "go.temporal.io/sdk/worker.Worker"
	ClientType  = "go.temporal.io/sdk/client.Client"
	WorkflowPkg = "go.temporal.io/sdk/workflow"

	WorkflowCtxRe = "go\\.temporal\\.io/sdk/(workflow|internal)\\.Context"
	ActivityCtx   = "context.Context"

	RegisterActivity            = "RegisterActivity"
	RegisterActivityWithOptions = "RegisterActivityWithOptions"
	RegisterWorkflow            = "RegisterWorkflow"

	ExecuteActivity = "ExecuteActivity"
	ExecuteWorkflow = "ExecuteWorkflow"
)

var WorkflowCtx = regexp.MustCompile(WorkflowCtxRe)
