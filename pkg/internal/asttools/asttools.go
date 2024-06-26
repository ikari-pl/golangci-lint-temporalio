package asttools

import (
	"fmt"
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
// Returns if the type is serializable, and if not, why.
func IsSerializable(t types.Type) (bool, string) {
	// if the type has a custom Marshaler, it means the author of the type
	// knows how to serialize it, so we assume it's serializable

	// we cannot check if it implements Marshaler directly. It's from the `types` package,
	// and not from reflect (it defines a type at compile time, not at runtime),
	// so we have to check if it's a named type, and if it has a method called MarshalJSON
	if named, ok := t.(*types.Named); ok {
		for i := 0; i < named.NumMethods(); i++ {
			m := named.Method(i)
			if m.Name() == "MarshalJSON" {
				return true, "implements MarshalJSON"
			}
			if m.Name() == "ProtoMessage" {
				return true, "is a protobuf message"
			}
		}
	} else if named, ok := t.Underlying().(*types.Named); ok {
		return IsSerializable(named)
	}

	switch t := t.(type) {
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if is, why := IsSerializable(f.Type()); !is {
				return false, fmt.Sprintf("field %s (%s) is not serializable,\n\treason: %s", f.Name(), f.Type().String(), why)
			}
			// check if field serializes to json:
			// get json tag, it can be something like `json:"name,omitempty"`
			t := reflect.StructTag(t.Tag(i))
			jsonTag := t.Get("json")
			switch jsonTag {
			case "-":
				return false, fmt.Sprintf("field %s is not serializable,\n\treason: field is marked with json:\"-\"", f.Name())
			case "":
				if ast.IsExported(f.Name()) {
					if is, why := IsSerializable(f.Type()); !is {
						return false, fmt.Sprintf("field %s (%s) is not serializable,\n\treason: %s", f.Name(), f.Type().String(), why)
					}
					return true, ""
				}
				return false, fmt.Sprintf("field %s is not serializable,\n\treason: field is not exported", f.Name())
			}
		}
		return true, ""
	case *types.Pointer:
		return IsSerializable(t.Elem())
	case *types.Named:
		return IsSerializable(t.Underlying())
	case *types.Basic:
		return true, ""
	case *types.Slice:
		return IsSerializable(t.Elem())
	case *types.Array:
		return IsSerializable(t.Elem())
	case *types.Map:
		if is, why := IsSerializable(t.Key()); !is {
			// ideally we should check if map key is a basic type, probably
			return false, fmt.Sprintf("map key (%s) is not serializable,\n\treason: %s", t.Key().String(), why)
		}
		return IsSerializable(t.Elem())
	case *types.Interface:
		// if it's an interface, we can't know what it is, so make an optimistic assumption
		return true, ""
	default:
		// if we don't know what it is, assume it's not serializable
		return false, fmt.Sprintf("type %s is not serializable (most likely)", t.String())
	}
}
