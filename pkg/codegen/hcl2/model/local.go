package model

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type LocalVariable struct {
	Syntax *hclsyntax.Attribute

	Type  Type
	Value Expression
}

func (lv *LocalVariable) SyntaxNode() hclsyntax.Node {
	return lv.Syntax
}

func (*LocalVariable) isNode() {}
