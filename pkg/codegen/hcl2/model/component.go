package model

import "github.com/hashicorp/hcl/v2/hclsyntax"

type Component struct {
	Syntax *hclsyntax.Block

	InputTypes  map[string]Type
	OutputTypes map[string]Type

	Children []*Resource
	Locals   []*LocalVariable
}
