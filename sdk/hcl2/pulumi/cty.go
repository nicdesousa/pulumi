package pulumi

import (
	"reflect"

	pschema "github.com/pulumi/pulumi/pkg/codegen/schema"
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/zclconf/go-cty/cty"
)

var assetType = reflect.TypeOf((*resource.Asset)(nil)).Elem()
var archiveType = reflect.TypeOf((*resource.Archive)(nil)).Elem()
var assetCapsule = cty.Capsule("Asset", assetType)
var archiveCapsule = cty.Capsule("Archive", archiveType)

func makeType(t pschema.Type) cty.Type {
	switch t := t.(type) {
	case *pschema.ArrayType:
		return cty.List(makeType(t.ElementType))
	case *pschema.MapType:
		return cty.Map(makeType(t.ElementType))
	case *pschema.ObjectType:
		attrTypes := map[string]cty.Type{}
		for _, prop := range t.Properties {
			attrTypes[prop.Name] = makeType(prop.Type)
		}
		return cty.Object(attrTypes)
	case *pschema.TokenType:
		if t.UnderlyingType != nil {
			return makeType(t.UnderlyingType)
		}
		return cty.DynamicPseudoType
	case *pschema.UnionType:
		return cty.DynamicPseudoType
	default:
		switch t {
		case pschema.BoolType:
			return cty.Bool
		case pschema.IntType, pschema.NumberType:
			return cty.Number
		case pschema.StringType:
			return cty.String
		case pschema.ArchiveType:
			return archiveCapsule
		case pschema.AssetType:
			return assetCapsule
		case pschema.AnyType:
			return cty.DynamicPseudoType
		}
	}
	return cty.DynamicPseudoType
}

type countIterator struct {
	current cty.Value
	limit   cty.Value
}

func newCountIterator(limit cty.Value) cty.ElementIterator {
	return &countIterator{
		current: cty.NumberIntVal(0),
		limit:   limit,
	}
}

func (it *countIterator) Next() bool {
	more := it.current.LessThan(it.limit).True()
	if more {
		it.current = it.current.Add(cty.NumberIntVal(1))
	}
	return more
}

func (it *countIterator) Element() (cty.Value, cty.Value) {
	return it.current, it.current
}
