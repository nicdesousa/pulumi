package model

import (
	"fmt"
	"strings"
)

// Type represents a datatype in the Pulumi Schema. Types created by this package are identical if they are
// equal values.
type Type interface {
	String() string

	isType()
}

type primitiveType int

const (
	boolType    primitiveType = 1
	intType     primitiveType = 2
	numberType  primitiveType = 3
	stringType  primitiveType = 4
	archiveType primitiveType = 5
	assetType   primitiveType = 6
	anyType     primitiveType = 7
)

func (t primitiveType) String() string {
	switch t {
	case boolType:
		return "boolean"
	case intType:
		return "integer"
	case numberType:
		return "number"
	case stringType:
		return "string"
	case archiveType:
		return "pulumi:pulumi:Archive"
	case assetType:
		return "pulumi:pulumi:Asset"
	case anyType:
		return "pulumi:pulumi:Any"
	default:
		panic("unknown primitive type")
	}
}

func (primitiveType) isType() {}

// IsPrimitiveType returns true if the given Type is a primitive type. The primitive types are bool, int, number,
// string, archive, asset, and any.
func IsPrimitiveType(t Type) bool {
	_, ok := t.(primitiveType)
	return ok
}

var (
	// BoolType represents the set of boolean values.
	BoolType Type = boolType
	// IntType represents the set of 32-bit integer values.
	IntType Type = intType
	// NumberType represents the set of IEEE754 double-precision values.
	NumberType Type = numberType
	// StringType represents the set of UTF-8 string values.
	StringType Type = stringType
	// ArchiveType represents the set of Pulumi Archive values.
	ArchiveType Type = archiveType
	// AssetType represents the set of Pulumi Asset values.
	AssetType Type = assetType
	// AnyType represents the complete set of values.
	AnyType Type = anyType
)

// OptionalType represents values of a particular type that are optional.
//
// Note: we could construct this out of an undefined type and a union type, but that seems awfully fancy for our
// purposes.
type OptionalType struct {
	// ElementType is the non-optional element type.
	ElementType Type
}

func (t *OptionalType) String() string {
	return fmt.Sprintf("optional(%v)", t.ElementType)
}

func (t *OptionalType) isType() {}

// OutputType represents eventual values that carry dependency information (e.g. resource output properties)
type OutputType struct {
	// ElementType is tne element type of the output.
	ElementType Type
}

func (t *OutputType) String() string {
	return fmt.Sprintf("output(%v)", t.ElementType)
}

func (t *OutputType) isType() {}

// PromiseType represents eventual values that do not carry dependency information (e.g invoke return values)
type PromiseType struct {
	// ElementType is tne element type of the promise.
	ElementType Type
}

func (t *PromiseType) String() string {
	return fmt.Sprintf("promise(%v)", t.ElementType)
}

func (t *PromiseType) isType() {}

// MapType represents maps from strings to particular element types.
type MapType struct {
	// ElementType is the element type of the map.
	ElementType Type
}

func (t *MapType) String() string {
	return fmt.Sprintf("map(%v)", t.ElementType)
}

func (*MapType) isType() {}

// ArrayType represents arrays of particular element types.
type ArrayType struct {
	// ElementType is the element type of the array.
	ElementType Type
}

func (t *ArrayType) String() string {
	return fmt.Sprintf("array(%v)", t.ElementType)
}

func (*ArrayType) isType() {}

// UnionType represents values that may be any one of a specified set of types.
type UnionType struct {
	// ElementTypes are the allowable types for the union type.
	ElementTypes []Type
}

func (t *UnionType) String() string {
	elements := make([]string, len(t.ElementTypes))
	for i, e := range t.ElementTypes {
		elements[i] = e.String()
	}
	return fmt.Sprintf("union(%s)", strings.Join(elements, ", "))
}

func (*UnionType) isType() {}

// ObjectType represents schematized maps from strings to particular types.
type ObjectType struct {
	// Properties records the types of the object's properties.
	Properties map[string]Type
}

func (t *ObjectType) String() string {
	var elements []string
	for k, v := range t.Properties {
		elements = append(elements, fmt.Sprintf("%s = %v", k, v))
	}

	return fmt.Sprintf("object({%s})", strings.Join(elements, ", "))
}

func (*ObjectType) isType() {}

// TokenType represents a type that is named by a type token.
type TokenType struct {
	// Token is the type's Pulumi type token.
	Token string

	// UnderlyingType is the underlying type named by the token.
	UnderlyingType Type
}

func (t *TokenType) String() string {
	return fmt.Sprintf("token(%s, %v)", t.Token, t.UnderlyingType)
}

func (*TokenType) isType() {}
