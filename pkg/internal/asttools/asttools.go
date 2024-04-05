package asttools

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"reflect"
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

// IsSerializable returns true if the given type is serializable to JSON.
// This is a very rough approximation, but it's good enough for our purposes.
func IsSerializable(t types.Type) bool {
	// if the type has a custom Marshaler, it means the author of the type
	// knows how to serialize it, so we assume it's serializable
	ut := t.Underlying()
	tt := reflect.TypeOf(ut)
	if tt.Implements(reflect.TypeOf((*json.Marshaler)(nil)).Elem()) {
		return true
	}
	if tt.Implements(reflect.TypeOf((json.Marshaler)(nil)).Elem()) {
		return true
	}

	switch t := t.(type) {
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if !IsSerializable(f.Type()) {
				return false
			}
			// check if field serializes to json:
			// get json tag, it can be something like `json:"name,omitempty"`
			t := reflect.StructTag(t.Tag(i))
			jsonTag := t.Get("json")
			switch jsonTag {
			case "-":
				return false
			case "":
				return ast.IsExported(f.Name()) && IsSerializable(f.Type())
			}
		}
		return true
	case *types.Pointer:
		return IsSerializable(t.Elem())
	case *types.Named:
		return IsSerializable(t.Underlying())
	case *types.Basic:
		return true
	case *types.Slice:
		return IsSerializable(t.Elem())
	case *types.Map:
		return IsSerializable(t.Key()) && IsSerializable(t.Elem())
	case *types.Interface:
		// if it's an interface, we can't know what it is, so make an optimistic assumption
		return true
	default:
		// if we don't know what it is, assume it's not serializable
		return false
	}
}
