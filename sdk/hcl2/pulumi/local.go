package pulumi

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type localState struct {
	*awaitable

	name         string
	expr         hcl.Expression
	dependencies []node

	diagnostics hcl.Diagnostics
	value       cty.Value
}

func newLocalState(name string, expr hcl.Expression) *localState {
	return &localState{
		awaitable: newAwaitable(),
		name:      name,
		expr:      expr,
	}
}

func (ls *localState) nodeName() string {
	return ls.name
}

func (ls *localState) prepare(ctx *programContext) hcl.Diagnostics {
	deps, diags := expressionDeps(ctx, ls.expr)
	ls.dependencies = deps
	return diags
}

func (ls *localState) evaluate(ctx *programContext) {
	result := uint32(awaitableResolved)

	defer func() {
		ls.fulfill(result)
	}()

	vars, funcs := map[string]cty.Value{}, map[string]function.Function{}
	for _, dep := range ls.dependencies {
		val, _, ok := dep.await(ctx)
		if !ok {
			result = awaitableCanceled
			return
		}
		name := dep.nodeName()
		if val.Type().Equals(funcCapsule) {
			funcs[name] = *(val.EncapsulatedValue().(*function.Function))
		} else {
			vars[name] = val
		}
	}
	evalContext := builtinEvalContext.NewChild()
	evalContext.Variables, evalContext.Functions = vars, funcs

	ls.value, ls.diagnostics = ls.expr.Value(evalContext)
	if ls.diagnostics.HasErrors() {
		result = awaitableRejected
	}
}

func (ls *localState) await(ctx *programContext) (cty.Value, hcl.Diagnostics, bool) {
	if ok := ls.awaitable.await(ctx); !ok {
		return cty.Value{}, ls.diagnostics, false
	}

	return ls.value, nil, true
}
