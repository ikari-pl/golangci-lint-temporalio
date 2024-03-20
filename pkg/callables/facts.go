package callables

import (
	"go/types"
)

type isWorkflow struct{}

func (i isWorkflow) AFact() {}

type isActivity struct{}

func (i isActivity) AFact() {}

type isWorkflowCall struct{}

func (i isWorkflowCall) AFact() {}

type isActivityCall struct{}

func (i isActivityCall) AFact() {}

type Callables struct {
	Workflows  []types.Object
	Activities []types.Object
}
