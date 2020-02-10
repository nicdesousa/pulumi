// Copyright 2016-2018, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pulumi

import (
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/pulumi/pulumi/pkg/util/contract"
	"github.com/zclconf/go-cty/cty"
)

var secretMark interface{} = &struct{}{}

// marshalValue converts the given cty.Value to a resource.PropertyMap.
func marshalValue(v cty.Value) resource.PropertyValue {
	switch {
	case v.IsNull():
		return resource.PropertyValue{}
	case !v.IsKnown():
		return resource.MakeComputed(resource.NewStringProperty(""))
	case v.HasMark(secretMark):
		unmarked, _ := v.Unmark()
		return resource.MakeSecret(marshalValue(unmarked))
	}

	t := v.Type()
	switch {
	case t.IsListType() || t.IsSetType() || t.IsTupleType():
		var arr []resource.PropertyValue
		it := v.ElementIterator()
		for it.Next() {
			_, e := it.Element()
			if !v.IsNull() {
				arr = append(arr, marshalValue(e))
			}
		}
		return resource.NewArrayProperty(arr)
	case t.IsMapType() || t.IsObjectType():
		m := resource.PropertyMap{}
		it := v.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			if !v.IsNull() {
				m[resource.PropertyKey(k.AsString())] = marshalValue(v)
			}
		}
		return resource.NewObjectProperty(m)
	case t.IsCapsuleType() && t.EncapsulatedType() == assetType:
		return resource.NewAssetProperty(v.EncapsulatedValue().(*resource.Asset))
	case t.IsCapsuleType() && t.EncapsulatedType() == archiveType:
		return resource.NewArchiveProperty(v.EncapsulatedValue().(*resource.Archive))
	case t == cty.Bool:
		return resource.NewBoolProperty(v.True())
	case t == cty.Number:
		f, _ := v.AsBigFloat().Float64()
		return resource.NewNumberProperty(f)
	case t == cty.String:
		return resource.NewStringProperty(v.AsString())
	default:
		contract.Failf("unexpected type: %v", t)
		return resource.PropertyValue{}
	}
}

// TODO: schema-driven unmarshaling

// unmarshalValue converts the given resource.PropertyValue to a cty.Value.
func unmarshalValue(v resource.PropertyValue) cty.Value {
	switch {
	case v.IsNull():
		return cty.NullVal(cty.DynamicPseudoType)
	case v.IsComputed() || v.IsOutput():
		return cty.UnknownVal(cty.DynamicPseudoType)
	case v.IsSecret():
		return unmarshalValue(v.SecretValue().Element).Mark(secretMark)
	case v.IsArray():
		arr := v.ArrayValue()
		if len(arr) == 0 {
			return cty.ListValEmpty(cty.DynamicPseudoType)
		}
		vals := make([]cty.Value, len(arr))
		for i := range arr {
			vals[i] = unmarshalValue(arr[i])
		}
		return cty.ListVal(vals)
	case v.IsObject():
		attrs := make(map[string]cty.Value)
		for k, e := range v.ObjectValue() {
			attrs[string(k)] = unmarshalValue(e)
		}
		return cty.ObjectVal(attrs)
	case v.IsAsset():
		return cty.CapsuleVal(assetCapsule, v.AssetValue())
	case v.IsArchive():
		return cty.CapsuleVal(archiveCapsule, v.ArchiveValue())
	case v.IsBool():
		return cty.BoolVal(v.BoolValue())
	case v.IsNumber():
		return cty.NumberFloatVal(v.NumberValue())
	case v.IsString():
		return cty.StringVal(v.StringValue())
	default:
		contract.Failf("unexpected type: %v", v.TypeString())
		return cty.Value{}
	}
}
