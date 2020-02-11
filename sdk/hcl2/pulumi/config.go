package pulumi

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type configState struct {
	*awaitable

	name         string
	defaultValue hcl.Expression
	dependencies []node

	diagnostics hcl.Diagnostics
	value       cty.Value
}

func newConfigState(name string, defaultValue hcl.Expression) *configState {
	return &configState{
		awaitable:    newAwaitable(),
		name:         name,
		defaultValue: expr,
	}
}

func (cs *configState) nodeName() string {
	return cs.name
}

func (cs *configState) prepare(ctx *programContext) hcl.Diagnostics {
	deps, diags := expressionDeps(ctx, cs.expr)
	cs.dependencies = deps
	return diags
}

func (cs *configState) evaluate(ctx *programContext) {
	result := uint32(awaitableResolved)

	defer func() {
		cs.fulfill(result)
	}()

	vars, funcs := map[string]cty.Value{}, map[string]function.Function{}
	for _, dep := range cs.dependencies {
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

	cs.value, cs.diagnostics = cs.expr.Value(evalContext)
	if cs.diagnostics.HasErrors() {
		result = awaitableRejected
	}
}

func (cs *configState) await(ctx *programContext) (cty.Value, hcl.Diagnostics, bool) {
	if ok := cs.awaitable.await(ctx); !ok {
		return cty.Value{}, cs.diagnostics, false
	}

	return cs.value, nil, true
}
