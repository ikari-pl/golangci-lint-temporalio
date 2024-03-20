package asttools

import (
	"go/ast"
	"go/types"
)

// IdentifierOf returns the *ast.Ident for the given *ast.Expr, whatever it is.
func IdentifierOf(e ast.Expr) *ast.Ident {
	switch e := e.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return IdentifierOf(e.Sel)
	case *ast.StarExpr:
		return IdentifierOf(e.X)
	case *ast.CallExpr:
		return IdentifierOf(e.Fun)
	// pointer to a function
	case *ast.FuncType:
		return IdentifierOf(e)
	// pointer to a struct (represented as unary AND of a composite literal)
	case *ast.UnaryExpr:
		return IdentifierOf(e.X)
	// continuation of pointer to a struct (represented as composite literal)
	case *ast.CompositeLit:
		return IdentifierOf(e.Type)
	default:
		return nil
	}
}

func IsSerializable(t types.Type) bool {
	switch t := t.(type) {
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if !IsSerializable(f.Type()) {
				return false
			}
		}
		return true
	case *types.Pointer:
		return IsSerializable(t.Elem())
	case *types.Named:
		return IsSerializable(t.Underlying())
	default:
		return true
	}
}
