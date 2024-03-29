package types

import (
	"go/ast"
	"go/token"
	"go/types"
)

// TemporalCall represents a detected Temporal.io workflow or activity invocation.
type TemporalCall struct {
	Pos      token.Pos
	FileName string
	CallName string
	Expr     *ast.CallExpr

	Type     TemporalIoCallType
	Callee   types.Object
	CallArgs []ast.Expr
}

type TemporalIoCallType int

const (
	NotSupported TemporalIoCallType = iota
	Workflow
	Activity
)
