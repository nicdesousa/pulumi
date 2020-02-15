package model

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type ConfigVariable struct {
	Syntax *hclsyntax.Block

	Type         Type
	DefaultValue Expression
}

func (cv *ConfigVariable) SyntaxNode() hclsyntax.Node {
	return cv.Syntax
}

func (*ConfigVariable) isNode() {}
