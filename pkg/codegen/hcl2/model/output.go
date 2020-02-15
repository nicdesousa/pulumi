package model

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type OutputVariable struct {
	Syntax *hclsyntax.Block

	Type  Type
	Value Expression
}

func (ov *OutputVariable) SyntaxNode() hclsyntax.Node {
	return ov.Syntax
}

func (*OutputVariable) isNode() {}
