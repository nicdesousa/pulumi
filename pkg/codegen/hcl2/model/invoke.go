package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func getInvokeToken(call *hclsyntax.FunctionCallExpr) (string, hcl.Range, bool) {
	if call.Name != "invoke" || len(call.Args) < 1 {
		return "", hcl.Range{}, false
	}
	literal, ok := call.Args[0].(*hclsyntax.LiteralValueExpr)
	if !ok {
		return "", hcl.Range{}, false
	}
	if literal.Val.Type() != cty.String {
		return "", hcl.Range{}, false
	}
	return literal.Val.AsString(), call.Args[0].Range(), true
}
