package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pulumi/pulumi/pkg/util/contract"
)

func bindExpression(syntax hclsyntax.Node) (Expression, hcl.Diagnostics) {
	switch syntax := syntax.(type) {
	case *hclsyntax.AnonSymbolExpr:
		return bindAnonSymbolExpression(syntax)
	case *hclsyntax.BinaryOpExpr:
		return bindBinaryOpExpression(syntax)
	case *hclsyntax.ConditionalExpr:
		return bindConditionalExpression(syntax)
	case *hclsyntax.ForExpr:
		return bindForExpression(syntax)
	case *hclsyntax.FunctionCallExpr:
		return bindFunctionCallExpression(syntax)
	case *hclsyntax.IndexExpr:
		return bindIndexExpression(syntax)
	case *hclsyntax.LiteralValueExpr:
		return bindLiteralValueExpression(syntax)
	case *hclsyntax.ObjectConsExpr:
		return bindObjectConsExpression(syntax)
	case *hclsyntax.RelativeTraversalExpr:
		return bindRelativeTraversalExpression(syntax)
	case *hclsyntax.ScopeTraversalExpr:
		return bindScopeTraversalExpression(syntax)
	case *hclsyntax.SplatExpr:
		return bindSplatExpression(syntax)
	case *hclsyntax.TemplateExpr:
		return bindTemplateExpression(syntax)
	case *hclsyntax.TemplateJoinExpr:
		return bindTemplateJoinExpression(syntax)
	case *hclsyntax.TemplateWrapExpr:
		return bindTemplateWrapExpression(syntax)
	case *hclsyntax.TupleConsExpr:
		return bindTupleConsExpression(syntax)
	case *hclsyntax.UnaryOpExpr:
		return bindUnaryOpExpression(syntax)
	default:
		contract.Failf("unexpected expression node of type %T (%v)", syntax, syntax.Range())
		return nil, nil
	}
}

func bindAnonSymbolExpression(syntax *hclsyntax.AnonSymbolExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindBinaryOpExpression(syntax *hclsyntax.BinaryOpExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindConditionalExpression(syntax *hclsyntax.ConditionalExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindForExpression(syntax *hclsyntax.ForExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindFunctionCallExpression(syntax *hclsyntax.FunctionCallExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindIndexExpression(syntax *hclsyntax.IndexExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindLiteralValueExpression(syntax *hclsyntax.LiteralValueExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindObjectConsExpression(syntax *hclsyntax.ObjectConsExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindRelativeTraversalExpression(syntax *hclsyntax.RelativeTraversalExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindScopeTraversalExpression(syntax *hclsyntax.ScopeTraversalExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindSplatExpression(syntax *hclsyntax.SplatExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindTemplateExpression(syntax *hclsyntax.TemplateExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindTemplateJoinExpression(syntax *hclsyntax.TemplateJoinExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindTemplateWrapExpression(syntax *hclsyntax.TemplateWrapExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindTupleConsExpression(syntax *hclsyntax.TupleConsExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}

func bindUnaryOpExpression(syntax *hclsyntax.UnaryOpExpr) (Expression, hcl.Diagnostics) {
	return &ErrorExpression{Syntax: syntax}, notYetImplemented(syntax)
}
