package pulumi

import (
	"github.com/d5/tengo/v2"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/util/contract"
	"github.com/zclconf/go-cty/cty"
)

type capsuleObject struct {
	tengo.ObjectImpl
	Value cty.Value
}

func convertValueToScriptObject(v cty.Value) tengo.Object {
	switch {
	case v.IsNull():
		return tengo.UndefinedValue
	case !v.IsKnown():
		contract.Failf("unexpected undefined value")
		return tengo.UndefinedValue
	}

	t := v.Type()
	switch {
	case t.IsListType() || t.IsSetType() || t.IsTupleType():
		var arr []tengo.Object
		it := v.ElementIterator()
		for it.Next() {
			_, e := it.Element()
			arr = append(arr, convertValueToScriptObject(e))
		}
		return &tengo.Array{Value: arr}
	case t.IsMapType() || t.IsObjectType():
		m := map[string]tengo.Object{}
		it := v.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			m[k.AsString()] = convertValueToScriptObject(v)
		}
		return &tengo.Map{Value: m}
	case t == cty.Bool:
		if v.True() {
			return tengo.TrueValue
		}
		return tengo.FalseValue
	case t == cty.Number:
		f, _ := v.AsBigFloat().Float64()
		return &tengo.Float{Value: f}
	case t == cty.String:
		return &tengo.String{Value: v.AsString()}
	case t.IsCapsuleType():
		return &capsuleObject{Value: v}
	default:
		contract.Failf("unexpected type: %v", t)
		return tengo.UndefinedValue
	}
}

func convertScriptObjectToValue(o tengo.Object) (cty.Value, error) {
	if o == nil {
		return cty.NullVal(cty.DynamicPseudoType), nil
	}

	switch o := o.(type) {
	case *tengo.Array, *tengo.ImmutableArray:
		it, arr := o.Iterate(), []cty.Value{}
		for it.Next() {
			v, err := convertScriptObjectToValue(it.Value())
			if err != nil {
				return cty.Value{}, err
			}
			arr = append(arr, v)
		}
		if len(arr) == 0 {
			return cty.ListValEmpty(cty.DynamicPseudoType), nil
		}
		return cty.ListVal(arr), nil
	case *tengo.Bool:
		return cty.BoolVal(o == tengo.TrueValue), nil
	case *tengo.BuiltinFunction, *tengo.CompiledFunction, *tengo.UserFunction:
		// TODO: error
		return cty.NullVal(cty.DynamicPseudoType), nil
	case *tengo.Bytes:
		bytes := o.Value
		if len(bytes) == 0 {
			return cty.ListValEmpty(cty.Number), nil
		}
		numbers := make([]cty.Value, len(bytes))
		for i, b := range bytes {
			numbers[i] = cty.NumberIntVal(int64(b))
		}
		return cty.ListVal(numbers), nil
	case *tengo.Char:
		return cty.NumberIntVal(int64(o.Value)), nil
	case *tengo.Error:
		return cty.Value{}, errors.New(o.String())
	case *tengo.Float:
		return cty.NumberFloatVal(o.Value), nil
	case *tengo.Int:
		return cty.NumberIntVal(o.Value), nil
	case *tengo.Map, *tengo.ImmutableMap:
		it, attrs := o.Iterate(), map[string]cty.Value{}
		for it.Next() {
			v, err := convertScriptObjectToValue(it.Value())
			if err != nil {
				return cty.Value{}, err
			}
			attrs[it.Key().String()] = v
		}
		return cty.ObjectVal(attrs), nil
	case *tengo.String:
		return cty.StringVal(o.Value), nil
	case *tengo.Time:
		return cty.NumberIntVal(o.Value.UnixNano()), nil
	case *tengo.Undefined:
		return cty.NullVal(cty.DynamicPseudoType), nil
	default:
		return cty.Value{}, errors.Errorf("unexpected script object of type %T", o)
	}
}
