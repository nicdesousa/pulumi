// Copyright 2016-2018, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pulumi

import (
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/pulumi/pulumi/pkg/resource/plugin"
	"github.com/pulumi/pulumi/pkg/util/contract"
	pulumirpc "github.com/pulumi/pulumi/sdk/proto/go"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

const (
	resourceRegistering           = awaitablePending
	resourceRegistrationSucceeded = awaitableResolved
	resourceRegistrationFailed    = awaitableRejected
	resourceRegistrationCanceled  = awaitableCanceled
)

type resourceState struct {
	*awaitable

	name   string
	token  string
	custom bool

	schema *resourceSchema
	decl   *resourceDecl

	parent               *resourceState
	dependencies         []node
	propertyDependencies map[string][]*resourceState

	allURNs     []string
	allIDs      []string
	allOutputs  []cty.Value
	diagnostics hcl.Diagnostics
}

func diagnosticsFromError(err error) hcl.Diagnostics {
	return []*hcl.Diagnostic{{Severity: hcl.DiagError, Summary: err.Error()}}
}

func newResourceState(name, token string, custom bool, schema *resourceSchema, decl *resourceDecl) *resourceState {
	return &resourceState{
		awaitable:            newAwaitable(),
		name:                 name,
		token:                token,
		custom:               custom,
		schema:               schema,
		decl:                 decl,
		propertyDependencies: map[string][]*resourceState{},
	}
}

func (rs *resourceState) nodeName() string {
	return rs.name
}

func (rs *resourceState) hasRange() bool {
	return rs.decl != nil && rs.decl.Options != nil && rs.decl.Options.Range != nil
}

func (rs *resourceState) urn() string {
	contract.Assert(!rs.hasRange())
	return rs.allURNs[0]
}

func (rs *resourceState) outputs() cty.Value {
	if !rs.hasRange() {
		return rs.allOutputs[0]
	}
	if len(rs.allOutputs) == 0 {
		return cty.ListValEmpty(cty.DynamicPseudoType)
	}
	return cty.ListVal(rs.allOutputs)
}

func (rs *resourceState) prepare(ctx *programContext) hcl.Diagnostics {
	// Decode the body of the resource config.
	impliedSchema := hcldec.ImpliedSchema(rs.schema.spec)
	content, diags := rs.decl.Config.Content(impliedSchema)
	if diags.HasErrors() {
		return diags
	}

	// Collect the resource's dependencies and ensure they are registered before the resource itself.
	deps := map[node]struct{}{}
	rs.propertyDependencies = map[string][]*resourceState{}
	for _, attr := range content.Attributes {
		attrDeps, attrDiags := expressionDeps(ctx, attr.Expr)
		diags = append(diags, attrDiags...)
		for _, dep := range attrDeps {
			deps[dep] = struct{}{}
			if resourceDep, ok := dep.(*resourceState); ok {
				rs.propertyDependencies[attr.Name] = append(rs.propertyDependencies[attr.Name], resourceDep)
			}
		}
	}
	for typ, blocks := range content.Blocks.ByType() {
		for _, block := range blocks {
			var blockSpec hcldec.Spec
			switch spec := rs.schema.spec[typ].(type) {
			case *hcldec.BlockListSpec:
				blockSpec = spec.Nested
			case *hcldec.BlockMapSpec:
				blockSpec = spec.Nested
			case *hcldec.BlockObjectSpec:
				blockSpec = spec.Nested
			case *hcldec.BlockSetSpec:
				blockSpec = spec.Nested
			case *hcldec.BlockSpec:
				blockSpec = spec.Nested
			case *hcldec.BlockTupleSpec:
				blockSpec = spec.Nested
			case hcldec.ObjectSpec:
				blockSpec = spec
			default:
				contract.Failf("unexpected block spec type %T", spec)
			}

			for _, v := range hcldec.Variables(block.Body, blockSpec) {
				depName := v.RootName()
				if depName == "range" {
					continue
				}
				dep, ok := ctx.nodes[depName]
				if !ok {
					diags = append(diags, unknownResource(depName, v.SourceRange()))
				} else {
					deps[dep] = struct{}{}
					if resourceDep, ok := dep.(*resourceState); ok {
						rs.propertyDependencies[typ] = append(rs.propertyDependencies[typ], resourceDep)
					}
				}
			}
		}
	}
	if rs.hasRange() {
		exprDeps, exprDiags := expressionDeps(ctx, rs.decl.Options.Range)
		diags = append(diags, exprDiags...)
		for _, dep := range exprDeps {
			deps[dep] = struct{}{}
		}
	}
	for dep := range deps {
		rs.dependencies = append(rs.dependencies, dep)
	}
	return diags
}

func (rs *resourceState) evaluate(ctx *programContext) {
	result := uint32(resourceRegistrationSucceeded)

	defer func() {
		rs.fulfill(result)
	}()

	var parentURN string
	if rs != ctx.stack {
		if rs.parent == nil {
			rs.parent = ctx.stack
		}
		_, _, ok := rs.parent.await(ctx)
		if !ok {
			result = resourceRegistrationCanceled
			return
		}
		parentURN = rs.parent.urn()
	}

	vars, funcs := map[string]cty.Value{}, map[string]function.Function{}
	for _, dep := range rs.dependencies {
		val, _, ok := dep.await(ctx)
		if !ok {
			result = resourceRegistrationCanceled
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

	// Convert dependency information.
	rpcPropertyDeps := make(map[string]*pulumirpc.RegisterResourceRequest_PropertyDependencies)
	for k, deps := range rs.propertyDependencies {
		var urns []string
		for _, dep := range deps {
			urns = append(urns, dep.allURNs...)
		}
		sort.Strings(urns)

		rpcPropertyDeps[k] = &pulumirpc.RegisterResourceRequest_PropertyDependencies{
			Urns: urns,
		}
	}
	var rpcDeps []string
	for _, dep := range rs.dependencies {
		if resourceDep, ok := dep.(*resourceState); ok {
			rpcDeps = append(rpcDeps, resourceDep.allURNs...)
		}
	}
	sort.Strings(rpcDeps)

	count, rangeIter := 1, newCountIterator(cty.NumberIntVal(1))
	if rs.hasRange() {
		rangeVal, diags := rs.decl.Options.Range.Value(evalContext)
		if diags.HasErrors() {
			rs.diagnostics, result = diags, resourceRegistrationFailed
			return
		}
		if rangeVal.IsKnown() {
			t := rangeVal.Type()
			switch {
			case t == cty.Number:
				bigCount := rangeVal.AsBigFloat()
				count64, _ := bigCount.Int64()
				count, rangeIter = int(count64), newCountIterator(cty.NumberIntVal(count64))
			case t.IsListType() || t.IsMapType() || t.IsObjectType():
				count, rangeIter = rangeVal.LengthInt(), rangeVal.ElementIterator()
			default:
				rs.diagnostics, result = hcl.Diagnostics{invalidRange(rs.decl.Options.Range.Range())}, resourceRegistrationFailed
				return
			}
		} else {
			// TODO: log the unknown count
		}
	}

	var wg sync.WaitGroup
	wg.Add(count)

	type singleResult struct {
		diags     hcl.Diagnostics
		succeeded bool
		response  *pulumirpc.RegisterResourceResponse
	}

	registerOne := func(idx int, key, value cty.Value) singleResult {
		defer wg.Done()

		myEvalCtx, myName := evalContext.NewChild(), rs.name
		if rs.hasRange() {
			myEvalCtx.Variables = map[string]cty.Value{
				"range": cty.ObjectVal(map[string]cty.Value{
					"key":   key,
					"value": value,
				}),
			}

			myName = fmt.Sprintf("%s-%d", myName, idx)
		}

		var inputs cty.Value
		if rs.decl == nil {
			inputs = cty.ObjectVal(map[string]cty.Value{})
		} else {
			ins, diags := hcldec.Decode(rs.decl.Config, rs.schema.spec, myEvalCtx)
			if diags.HasErrors() {
				return singleResult{diags: diags}
			}
			inputs = ins
		}
		inputProps := marshalValue(inputs).ObjectValue()

		// Marshal all properties for the RPC call.
		keepUnknowns := ctx.info.DryRun
		rpcProps, err := plugin.MarshalProperties(inputProps, plugin.MarshalOptions{KeepUnknowns: keepUnknowns})
		if err != nil {
			return singleResult{diags: diagnosticsFromError(err)}
		}

		resp, err := ctx.monitor.RegisterResource(ctx.cancel, &pulumirpc.RegisterResourceRequest{
			Type:   rs.token,
			Name:   myName,
			Parent: parentURN,
			Object: rpcProps,
			Custom: rs.custom,
			// Protect:
			Dependencies: rpcDeps,
			// Provider:
			PropertyDependencies: rpcPropertyDeps,
			// DeleteBeforeReplace:
			// ImportId:
			// CustomTimeouts:
			// IgnoreChanges:
			// Aliases:
		})
		if err != nil {
			return singleResult{diags: diagnosticsFromError(err)}
		}

		return singleResult{succeeded: true, response: resp}
	}

	results := make([]singleResult, count)
	for idx := 0; rangeIter.Next(); idx++ {
		key, value := rangeIter.Element()
		go func(idx int) {
			results[idx] = registerOne(idx, key, value)
		}(idx)
	}
	wg.Wait()

	for _, r := range results {
		if !r.succeeded {
			rs.diagnostics, result = append(rs.diagnostics, r.diags...), resourceRegistrationFailed
			continue
		}

		outprops, err := plugin.UnmarshalProperties(r.response.Object, plugin.MarshalOptions{KeepSecrets: true})
		if err != nil {
			rs.diagnostics, result = append(rs.diagnostics, diagnosticsFromError(err)...), resourceRegistrationFailed
			continue
		}

		urn, id := r.response.Urn, r.response.Id
		rs.allURNs, rs.allIDs = append(rs.allURNs, urn), append(rs.allIDs, id)

		outprops["urn"] = resource.NewStringProperty(urn)
		if rs.custom {
			if id != "" || !ctx.info.DryRun {
				outprops["id"] = resource.NewStringProperty(id)
			} else {
				outprops["id"] = resource.MakeComputed(resource.PropertyValue{})
			}
		}

		if rs.schema != nil {
			for _, prop := range rs.schema.pulumi.Properties {
				k := resource.PropertyKey(prop.Name)
				if _, has := outprops[k]; !has {
					if ctx.info.DryRun {
						outprops[k] = resource.MakeComputed(resource.PropertyValue{})
					} else {
						outprops[k] = resource.PropertyValue{}
					}
				}
			}
		}

		rs.allOutputs = append(rs.allOutputs, unmarshalValue(resource.NewObjectProperty(outprops)))
	}
}

func (rs *resourceState) registerOutputs(ctx *programContext, outputs map[string]cty.Value) hcl.Diagnostics {
	if !rs.awaitable.await(ctx) {
		return nil
	}

	outs := marshalValue(cty.ObjectVal(outputs)).ObjectValue()

	keepUnknowns := ctx.info.DryRun
	outsMarshalled, err := plugin.MarshalProperties(outs, plugin.MarshalOptions{KeepUnknowns: keepUnknowns})
	if err != nil {
		return diagnosticsFromError(err)
	}

	_, err = ctx.monitor.RegisterResourceOutputs(ctx.cancel, &pulumirpc.RegisterResourceOutputsRequest{
		Urn:     rs.urn(),
		Outputs: outsMarshalled,
	})
	if err != nil {
		return diagnosticsFromError(err)
	}
	return nil
}

func (rs *resourceState) await(ctx *programContext) (cty.Value, hcl.Diagnostics, bool) {
	if !rs.awaitable.await(ctx) {
		return cty.Value{}, rs.diagnostics, false
	}
	return rs.outputs(), nil, true
}
