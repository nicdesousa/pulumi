package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/pulumi/pulumi/pkg/util/contract"
)

func (b *binder) bindNode(node Node) hcl.Diagnostics {
	switch node := node.(type) {
	case *ConfigVariable:
		return b.bindConfigVariable(node)
	case *LocalVariable:
		return b.bindLocalVariable(node)
	case *Resource:
		return b.bindResource(node)
	case *OutputVariable:
		return b.bindOutputVariable(node)
	default:
		contract.Failf("unexpected node of type %T (%v)", node, node.SyntaxNode().Range())
		return nil
	}
}

func (b *binder) bindConfigVariable(node *ConfigVariable) hcl.Diagnostics {
	return notYetImplemented(node)
}

func (b *binder) bindLocalVariable(node *LocalVariable) hcl.Diagnostics {
	return notYetImplemented(node)
}

func (b *binder) bindResource(node *Resource) hcl.Diagnostics {
	return notYetImplemented(node)
}

func (b *binder) bindOutputVariable(node *OutputVariable) hcl.Diagnostics {
	return notYetImplemented(node)
}
