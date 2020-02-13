package pulumi

import (
	"encoding/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type configState struct {
	*awaitable

	name         string
	defaultValue hcl.Expression
	dependencies []node
	resourceDeps []*resourceState

	diagnostics hcl.Diagnostics
	value       cty.Value
}

func newConfigState(name string, defaultValue hcl.Expression) *configState {
	return &configState{
		awaitable:    newAwaitable(),
		name:         name,
		defaultValue: defaultValue,
	}
}

func (cs *configState) nodeName() string {
	return cs.name
}

func (cs *configState) resourceDependencies() []*resourceState {
	return cs.resourceDeps
}

func (cs *configState) prepare(ctx *programContext) hcl.Diagnostics {
	deps, diags := expressionDeps(ctx, cs.defaultValue)
	cs.dependencies = deps
	return diags
}

func (cs *configState) evaluate(ctx *programContext) {
	result := uint32(awaitableResolved)

	defer func() {
		cs.fulfill(result)
	}()

	if stringVal, ok := ctx.info.Config[ctx.info.Project+":"+cs.name]; ok {
		var val interface{}
		if err := json.Unmarshal([]byte(stringVal), &val); err != nil {
			result, cs.diagnostics = awaitableRejected, diagnosticsFromError(err)
			return
		}

		cs.value = unmarshalValue(resource.NewPropertyValue(val))
		return
	}

	if cs.defaultValue == nil {
		result, cs.diagnostics = awaitableRejected, diagnosticsFromError(errors.Errorf("missing required config variable %s", cs.name))
		return
	}

	vars, funcs := map[string]cty.Value{}, map[string]function.Function{}
	for _, dep := range cs.dependencies {
		val, _, ok := dep.await(ctx)
		if !ok {
			result = awaitableCanceled
			return
		}

		cs.resourceDeps = append(cs.resourceDeps, dep.resourceDependencies()...)

		name := dep.nodeName()
		if val.Type().Equals(funcCapsule) {
			funcs[name] = *(val.EncapsulatedValue().(*function.Function))
		} else {
			vars[name] = val
		}
	}
	evalContext := ctx.evalContext.NewChild()
	evalContext.Variables, evalContext.Functions = vars, funcs

	cs.value, cs.diagnostics = cs.defaultValue.Value(evalContext)
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
