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
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type outputState struct {
	name         string
	expr         hcl.Expression
	dependencies []node
}

func (os *outputState) prepare(ctx *programContext) hcl.Diagnostics {
	// Collect the resource's dependencies and ensure they are registered before the resource itself.
	deps, diags := expressionDeps(ctx, os.expr)
	os.dependencies = deps
	return diags
}

func (os *outputState) evaluate(ctx *programContext) (cty.Value, hcl.Diagnostics) {
	vars, funcs := map[string]cty.Value{}, map[string]function.Function{}
	for _, dep := range os.dependencies {
		val, _, ok := dep.await(ctx)
		if !ok {
			return cty.UnknownVal(cty.DynamicPseudoType), nil
		}
		name := dep.nodeName()
		if val.Type().Equals(funcCapsule) {
			funcs[name] = *(val.EncapsulatedValue().(*function.Function))
		} else {
			vars[name] = val
		}
	}

	evalContext := ctx.evalContext.NewChild()
	evalContext.Variables, evalContext.Functions = vars, funcs

	return os.expr.Value(evalContext)
}
