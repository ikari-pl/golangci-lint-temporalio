package callables

import (
	"go/token"
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

// TemporalCall represents a detected Temporal.io workflow or activity invocation.
type TemporalCall struct {
	Pos      token.Pos
	FileName string
	CallName string
}

type Callables struct {
	Workflows  []types.Object
	Activities []types.Object
}
