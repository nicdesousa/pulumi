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

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/net/context"
)

// RunInfo contains all the metadata about a run request.
type RunInfo struct {
	Project     string
	Stack       string
	Config      map[string]string
	Parallel    int
	DryRun      bool
	MonitorAddr string
	EngineAddr  string
}

func Run(ctx context.Context, sources map[string][]byte, info RunInfo) (map[string]*hcl.File, hcl.Diagnostics) {
	// Validate some properties.
	if info.Project == "" {
		return nil, diagnosticsFromError(errors.Errorf("missing project name"))
	} else if info.Stack == "" {
		return nil, diagnosticsFromError(errors.New("missing stack name"))
	} else if info.MonitorAddr == "" {
		return nil, diagnosticsFromError(errors.New("missing resource monitor RPC address"))
	} else if info.EngineAddr == "" {
		return nil, diagnosticsFromError(errors.New("missing engine RPC address"))
	}

	pctx, err := newProgramContext(ctx, info)
	if err != nil {
		return nil, diagnosticsFromError(err)
	}

	// Parse the sources.
	for path, contents := range sources {
		if err := pctx.addFile(path, contents); err != nil {
			return pctx.parser.Files(), diagnosticsFromError(err)
		}
	}

	var diags hcl.Diagnostics

	// Prepare each resource.
	var resourceNames []string
	for n := range pctx.resources {
		resourceNames = append(resourceNames, n)
	}
	sort.Strings(resourceNames)
	for _, name := range resourceNames {
		resourceDiags := pctx.resources[name].prepare(pctx)
		diags = append(diags, resourceDiags...)
	}

	// Prepare each output
	var outputNames []string
	for n := range pctx.outputs {
		outputNames = append(outputNames, n)
	}
	sort.Strings(outputNames)
	for _, name := range outputNames {
		outputDiags := pctx.outputs[name].prepare(pctx)
		diags = append(diags, outputDiags...)
	}

	if diags.HasErrors() {
		return pctx.parser.Files(), diags
	}

	// Create a root stack resource that we'll parent everything to.
	pctx.stack = newResourceState(fmt.Sprintf("%s-%s", info.Project, info.Stack), "pulumi:pulumi:Stack", false, nil, nil)
	pctx.stack.register(pctx)

	// Kick off resource registrations.
	for _, r := range pctx.resources {
		go r.register(pctx)
	}

	// Await all resource registrations.
	for _, r := range pctx.resources {
		r.await(pctx)
		if r.state == resourceRegistrationFailed {
			diags = append(diags, r.diagnostics...)
		}
	}

	// Evaluate and register outputs.
	outputs := map[string]cty.Value{}
	for _, o := range pctx.outputs {
		val, valDiags := o.evaluate(pctx)
		diags = append(diags, valDiags...)
		outputs[o.name] = val
	}

	outDiags := pctx.stack.registerOutputs(pctx, outputs)
	diags = append(diags, outDiags...)

	return pctx.parser.Files(), diags
}
