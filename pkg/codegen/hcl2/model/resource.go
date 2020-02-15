package model

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type Resource struct {
	Syntax *hclsyntax.Block

	InputType  Type
	OutputType Type

	Inputs Expression
	Range  Expression

	// TODO: Resource options
}

func (r *Resource) SyntaxNode() hclsyntax.Node {
	return r.Syntax
}

func (*Resource) isNode() {}

// bind from syntax + schema
