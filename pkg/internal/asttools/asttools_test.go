package asttools

import (
	"go/types"
	"testing"
	"time"
)

type TestStruct struct {
	A int
	B string
	T time.Time
}

func TestIsSerializable(t *testing.T) {
	errorType := types.NewNamed(types.NewTypeName(0, nil, "Error", nil), types.NewStruct([]*types.Var{
		types.NewField(0, nil, "message", types.Typ[types.String], false),
	}, nil), nil)
	errorType.AddMethod(types.NewFunc(0, nil, "Error", types.NewSignatureType(
		nil,
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(0, nil, "", types.Typ[types.String]),
		),
		false,
	)))
	timeType := types.NewNamed(types.NewTypeName(0, nil, "Time", nil), types.NewStruct([]*types.Var{
		types.NewField(0, nil, "t", types.Typ[types.Int64], false),
	}, nil), nil)
	timeType.AddMethod(types.NewFunc(0, nil, "MarshalJSON", types.NewSignatureType(
		types.NewVar(0, nil, "t", timeType),
		nil,
		nil,
		types.NewTuple(
			types.NewVar(0, nil, "", types.NewPointer(types.Typ[types.Byte])),
		),
		types.NewTuple(
			// returns bytes and error
			types.NewVar(0, nil, "", types.NewSlice(types.Typ[types.Byte])),
			types.NewVar(0, nil, "", errorType),
		),
		false,
	)))

	shouldBeTrue := []types.Type{
		types.Typ[types.String],
		types.Typ[types.Int],
		types.Typ[types.Float32],
		types.Typ[types.Float64],
		types.Typ[types.Bool],
		types.NewSlice(types.Typ[types.Int]),
		types.NewArray(types.Typ[types.Int], 10),
		types.NewArray(types.Typ[types.Byte], 16),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewPointer(types.Typ[types.Int]),
		timeType,
		types.NewNamed(types.NewTypeName(0, nil, "TestStruct", nil), types.NewStruct([]*types.Var{
			types.NewField(0, nil, "A", types.Typ[types.Int], false),
			types.NewField(0, nil, "B", types.Typ[types.String], false),
			types.NewField(0, nil, "T", timeType, false),
		}, nil), nil),
	}
	for _, typ := range shouldBeTrue {
		t.Run("serializable types: "+typ.String(), func(t *testing.T) {
			if is, why := IsSerializable(typ); !is {
				t.Errorf("expected type %v to be serializable, wasn't: %s", typ, why)
			}
		})
	}
	shouldBeFalse := []types.Type{
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
	}
	for _, typ := range shouldBeFalse {
		t.Run("non-serializable types: "+typ.String(), func(t *testing.T) {
			if is, _ := IsSerializable(typ); is {
				t.Errorf("expected type %v to not be serializable", typ)
			}
		})
	}
}
