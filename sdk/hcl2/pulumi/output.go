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
)

type outputState struct {
	name         string
	expr         hcl.Expression
	dependencies []*resourceState
}

func (os *outputState) prepare(ctx *programContext) hcl.Diagnostics {
	// Collect the resource's dependencies and ensure they are registered before the resource itself.
	deps, diags := expressionDeps(ctx, os.expr)
	os.dependencies = deps
	return diags
}

func (os *outputState) evaluate(ctx *programContext) (cty.Value, hcl.Diagnostics) {
	vars := map[string]cty.Value{}
	for _, dep := range os.dependencies {
		if !dep.await(ctx) {
			return cty.UnknownVal(cty.DynamicPseudoType), nil
		}
		vars[dep.name] = dep.outputs()
	}

	return os.expr.Value(&hcl.EvalContext{Variables: vars})
}
