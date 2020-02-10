package pulumi

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pkg/errors"
	pulumirpc "github.com/pulumi/pulumi/sdk/proto/go"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type configVar struct {
	Type string `hcl:"type,label"`

	Default *cty.Value `hcl:"default,attr"`
}

type config struct {
	Vars map[string]*configVar `hcl:",remain"`
}

type packageDecl struct {
	NameOrPath string `hcl:"name,label"`
	Path       string `hcl:"path,label"`

	Config hcl.Body `hcl:",remain"`
}

type alias struct {
	URN     hcl.Expression `hcl:"urn,attr"`
	Name    hcl.Expression `hcl:"name,attr"`
	Type    hcl.Expression `hcl:"type,attr"`
	Parent  hcl.Expression `hcl:"parent,attr"`
	Stack   hcl.Expression `hcl:"stack,attr"`
	Project hcl.Expression `hcl:"project,attr"`
}

//type invokeOptions struct {
//	Parent   string `hcl:"parent,attr"`
//	Provider string `hcl:"provider,attr"`
//}

type timeouts struct {
	Create int `hcl:"create,attr"`
	Update int `hcl:"update,attr"`
	Delete int `hcl:"delete,attr"`
}

type resourceOptions struct {
	Range               hcl.Expression `hcl:"range,attr"`
	Parent              *string        `hcl:"parent,attr"`
	DependsOn           *[]string      `hcl:"dependsOn,attr"`
	Protect             *bool          `hcl:"protect,attr"`
	Provider            *string        `hcl:"provider,attr"`
	Providers           hcl.Expression `hcl:"providers,attr"`
	DeleteBeforeReplace *bool          `hcl:"deleteBeforeReplace,attr"`
	Import              *string        `hcl:"import,attr"`
	Timeouts            *timeouts      `hcl:"timeouts,block"`
	IgnoreChanges       *[]string      `hcl:"ignoreChanges,attr"`
	Aliases             []*alias       `hcl:"alias,block"`
}

//type callDecl struct {
//	Name string `hcl:"name,label"`
//
//	Options *invokeOptions `hcl:"options,block"`
//
//	Config hcl.Body `hcl:",remain"`
//}

type resourceDecl struct {
	Name string `hcl:"name,label"`
	Type string `hcl:"type,label"`

	Options *resourceOptions `hcl:"options,block"`

	Config hcl.Body `hcl:",remain"`
}

type outputsDecl struct {
	Vars map[string]hcl.Expression `hcl:",remain"`
}

type toplevel struct {
	Configs   []config       `hcl:"config,block"`
	Packages  []packageDecl  `hcl:"package,block"`
	Resources []resourceDecl `hcl:"resource,block"`
	Outputs   []outputsDecl  `hcl:"outputs,block"`

	Locals hcl.Body `hcl:",remain"`
}

type programContext struct {
	cancel context.Context
	info   RunInfo

	monitor pulumirpc.ResourceMonitorClient
	engine  pulumirpc.EngineClient

	parser    *hclparse.Parser
	schemae   map[string]*packageSchema
	resources map[string]*resourceState
	outputs   map[string]*outputState

	stack *resourceState
}

func newProgramContext(cancel context.Context, info RunInfo) (*programContext, error) {
	// Connect to the gRPC endpoints if we have addresses for them.
	var monitorConn *grpc.ClientConn
	var monitor pulumirpc.ResourceMonitorClient
	if addr := info.MonitorAddr; addr != "" {
		conn, err := grpc.Dial(info.MonitorAddr, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrap(err, "connecting to resource monitor over RPC")
		}
		monitorConn = conn
		monitor = pulumirpc.NewResourceMonitorClient(monitorConn)
	}

	var engineConn *grpc.ClientConn
	var engine pulumirpc.EngineClient
	if addr := info.EngineAddr; addr != "" {
		conn, err := grpc.Dial(info.EngineAddr, grpc.WithInsecure())
		if err != nil {
			return nil, errors.Wrap(err, "connecting to engine over RPC")
		}
		engineConn = conn
		engine = pulumirpc.NewEngineClient(engineConn)
	}

	//	if info.Mocks != nil {
	//		monitor = &mockMonitor{project: info.Project, stack: info.Stack, mocks: info.Mocks}
	//		engine = &mockEngine{}
	//	}

	return &programContext{
		cancel:    cancel,
		info:      info,
		monitor:   monitor,
		engine:    engine,
		parser:    hclparse.NewParser(),
		schemae:   map[string]*packageSchema{},
		resources: map[string]*resourceState{},
		outputs:   map[string]*outputState{},
	}, nil
}

func (ctx *programContext) ensureSchema(pkgName string) (*packageSchema, error) {
	schema, ok := ctx.schemae[pkgName]
	if ok {
		return schema, nil
	}

	schema, err := loadSchema(pkgName)
	if err != nil {
		return nil, err
	}
	ctx.schemae[pkgName] = schema
	return schema, nil
}

func (ctx *programContext) addFile(path string, contents []byte) hcl.Diagnostics {
	file, diags := ctx.parser.ParseHCL(contents, path)
	if diags.HasErrors() {
		return diags
	}

	var raw toplevel
	if diags = gohcl.DecodeBody(file.Body, nil, &raw); diags.HasErrors() {
		return diags
	}

	for _, r := range raw.Resources {
		pkgName, _, _ := decomposeToken(r.Type)
		pkgSchema, err := ctx.ensureSchema(pkgName)
		if err != nil {
			return diagnosticsFromError(err)
		}
		resourceSchema, ok := pkgSchema.resources[r.Type]
		if !ok {
			return diagnosticsFromError(errors.Errorf("unknown resource type %s", r.Type))
		}

		if r.Name == "range" {
			return diagnosticsFromError(errors.Errorf("resource may not be named 'range'", r.Name))
		}
		if _, ok := ctx.resources[r.Name]; ok {
			return diagnosticsFromError(errors.Errorf("duplicate resource %s", r.Name))
		}

		decl := r
		ctx.resources[r.Name] = newResourceState(r.Name, resourceSchema.pulumi.Token, true, resourceSchema, &decl)
	}

	for _, o := range raw.Outputs {
		for name, expr := range o.Vars {
			if _, ok := ctx.outputs[name]; ok {
				return diagnosticsFromError(errors.Errorf("duplicate output %s", name))
			}
			ctx.outputs[name] = &outputState{name: name, expr: expr}
		}
	}

	return nil
}

func expressionDeps(ctx *programContext, expr hcl.Expression) ([]*resourceState, hcl.Diagnostics) {
	var deps []*resourceState
	var diags hcl.Diagnostics
	for _, v := range expr.Variables() {
		depName := v.RootName()
		if depName == "range" {
			continue
		}
		dep, ok := ctx.resources[depName]
		if !ok {
			diags = append(diags, unknownResource(depName, v.SourceRange()))
		} else {
			deps = append(deps, dep)
		}
	}
	return deps, diags
}
