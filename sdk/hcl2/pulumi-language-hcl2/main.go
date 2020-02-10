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

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	pbempty "github.com/golang/protobuf/ptypes/empty"
	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/pulumi/pulumi/pkg/util/cmdutil"
	"github.com/pulumi/pulumi/pkg/util/contract"
	"github.com/pulumi/pulumi/pkg/util/logging"
	"github.com/pulumi/pulumi/pkg/util/rpcutil"
	"github.com/pulumi/pulumi/pkg/version"
	phcl2 "github.com/pulumi/pulumi/sdk/hcl2/pulumi"
	pulumirpc "github.com/pulumi/pulumi/sdk/proto/go"
)

// Launches the language host, which in turn fires up an RPC server implementing the LanguageRuntimeServer endpoint.
func main() {
	var tracing string
	flag.StringVar(&tracing, "tracing", "", "Emit tracing to a Zipkin-compatible tracing endpoint")

	flag.Parse()
	args := flag.Args()
	logging.InitLogging(false, 0, false)
	cmdutil.InitTracing("pulumi-language-hcl2", "pulumi-language-hcl2", tracing)

	// Pluck out the engine so we can do logging, etc.
	if len(args) == 0 {
		cmdutil.Exit(errors.New("missing required engine RPC address argument"))
	}
	engineAddress := args[0]

	// Fire up a gRPC server, letting the kernel choose a free port.
	port, done, err := rpcutil.Serve(0, nil, []func(*grpc.Server) error{
		func(srv *grpc.Server) error {
			host := newLanguageHost(engineAddress, tracing)
			pulumirpc.RegisterLanguageRuntimeServer(srv, host)
			return nil
		},
	}, nil)
	if err != nil {
		cmdutil.Exit(errors.Wrapf(err, "could not start language host RPC server"))
	}

	// Otherwise, print out the port so that the spawner knows how to reach us.
	fmt.Printf("%d\n", port)

	// And finally wait for the server to stop serving.
	if err := <-done; err != nil {
		cmdutil.Exit(errors.Wrapf(err, "language host RPC stopped serving"))
	}
}

// hcl2LanguageHost implements the LanguageRuntimeServer interface for use as an API endpoint.
type hcl2LanguageHost struct {
	engineAddress string
	tracing       string
}

func newLanguageHost(engineAddress, tracing string) pulumirpc.LanguageRuntimeServer {
	return &hcl2LanguageHost{
		engineAddress: engineAddress,
		tracing:       tracing,
	}
}

// GetRequiredPlugins computes the complete set of anticipated plugins required by a program.
func (host *hcl2LanguageHost) GetRequiredPlugins(ctx context.Context,
	req *pulumirpc.GetRequiredPluginsRequest) (*pulumirpc.GetRequiredPluginsResponse, error) {
	return &pulumirpc.GetRequiredPluginsResponse{}, nil
}

// RPC endpoint for LanguageRuntimeServer::Run
func (host *hcl2LanguageHost) Run(ctx context.Context, req *pulumirpc.RunRequest) (*pulumirpc.RunResponse, error) {
	// Read in all sources in the working directory.
	sourcePaths, err := filepath.Glob(filepath.Join(".", "*.pp"))
	if err != nil {
		return nil, err
	}
	sources := map[string][]byte{}
	for _, path := range sourcePaths {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sources[path] = contents
	}

	files, diags := phcl2.Run(ctx, sources, phcl2.RunInfo{
		Project:     req.GetProject(),
		Stack:       req.GetStack(),
		Config:      req.GetConfig(),
		DryRun:      req.GetDryRun(),
		Parallel:    int(req.GetParallel()),
		MonitorAddr: req.GetMonitorAddress(),
		EngineAddr:  host.engineAddress,
	})

	errResult := ""
	if len(diags) > 0 {
		err := hcl.NewDiagnosticTextWriter(os.Stderr, files, 0, true).WriteDiagnostics(diags)
		contract.IgnoreError(err)

		if diags.HasErrors() {
			errResult = "program failed"
		}
	}

	return &pulumirpc.RunResponse{Error: errResult}, nil
}

func (host *hcl2LanguageHost) GetPluginInfo(ctx context.Context, req *pbempty.Empty) (*pulumirpc.PluginInfo, error) {
	return &pulumirpc.PluginInfo{
		Version: version.Version,
	}, nil
}
