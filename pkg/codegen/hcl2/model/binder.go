package model

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pulumi/pulumi/pkg/codegen/hcl2/syntax"
	"github.com/pulumi/pulumi/pkg/codegen/schema"
	"github.com/pulumi/pulumi/pkg/resource/plugin"
	"github.com/pulumi/pulumi/pkg/tokens"
)

type binder struct {
	host plugin.Host

	packageSchemas map[string]*schema.Package
	nodes          map[string]Node
}

func BindProgram(files []*syntax.File, host plugin.Host) (*Program, hcl.Diagnostics, error) {
	if host == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, nil, err
		}
		ctx, err := plugin.NewContext(nil, nil, nil, nil, cwd, nil, nil)
		if err != nil {
			return nil, nil, err
		}
		host = ctx.Host
	}

	b := &binder{
		host:           host,
		packageSchemas: map[string]*schema.Package{},
		nodes:          map[string]Node{},
	}

	var diagnostics hcl.Diagnostics

	// Sort files in source order, then declare all top-level nodes in each.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	for _, f := range files {
		diagnostics = append(diagnostics, b.declareNodes(f)...)
	}

	// Sort nodes in source order so downstream operations are deterministic.
	var nodes []Node
	for _, n := range b.nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		ir, jr := nodes[i].SyntaxNode().Range(), nodes[j].SyntaxNode().Range()
		return ir.Filename < jr.Filename || ir.Start.Byte < jr.Start.Byte
	})

	// Load referenced package schemas and bind node types.
	for _, n := range nodes {
		if err := b.loadReferencedPackageSchemas(n); err != nil {
			return nil, nil, err
		}
	}

	// Now bind node bodies.
	for _, n := range nodes {
		diagnostics = append(diagnostics, b.bindNode(n)...)
	}

	return &Program{
		Nodes: nodes,
		files: files,
	}, diagnostics, nil
}

func (b *binder) declareNodes(file *syntax.File) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics

	// Declare blocks (config, resources, outputs), then attributes (locals)
	for _, block := range sourceOrderBlocks(file.Body.Blocks) {
		switch block.Type {
		case "config":
			if len(block.Labels) != 0 {
				diagnostics = append(diagnostics, labelsErrorf(block, "config blocks do not support labels"))
			}

			for _, attr := range sourceOrderAttributes(block.Body.Attributes) {
				diagnostics = append(diagnostics, errorf(attr.Range(), "unsupported attribute %q in config block", attr.Name))
			}

			for _, variable := range sourceOrderBlocks(block.Body.Blocks) {
				if len(variable.Labels) > 1 {
					diagnostics = append(diagnostics, labelsErrorf(block, "config variables must have no more than one label"))
				}

				diagnostics = append(diagnostics, b.declareNode(variable.Type, &ConfigVariable{
					Syntax: variable,
				})...)
			}
		case "resource":
			if len(block.Labels) != 2 {
				diagnostics = append(diagnostics, labelsErrorf(block, "resource variables must have exactly two labels"))
			}

			diagnostics = append(diagnostics, b.declareNode(block.Labels[0], &Resource{
				Syntax: block,
			})...)
		case "outputs":
			if len(block.Labels) != 0 {
				diagnostics = append(diagnostics, labelsErrorf(block, "outputs blocks do not support labels"))
			}

			for _, attr := range sourceOrderAttributes(block.Body.Attributes) {
				diagnostics = append(diagnostics, errorf(attr.Range(), "unsupported attribute %q in outputs block", attr.Name))
			}

			for _, variable := range sourceOrderBlocks(block.Body.Blocks) {
				if len(variable.Labels) > 1 {
					diagnostics = append(diagnostics, labelsErrorf(block, "output variables must have no more than one label"))
				}

				diagnostics = append(diagnostics, b.declareNode(variable.Type, &OutputVariable{
					Syntax: variable,
				})...)
			}
		}
	}

	for _, attr := range sourceOrderAttributes(file.Body.Attributes) {
		diagnostics = append(diagnostics, b.declareNode(attr.Name, &LocalVariable{
			Syntax: attr,
		})...)
	}

	return diagnostics
}

func (b *binder) declareNode(name string, n Node) hcl.Diagnostics {
	if existing, ok := b.nodes[name]; ok {
		return hcl.Diagnostics{errorf(existing.SyntaxNode().Range(), "%q already declared", name)}
	}

	b.nodes[name] = n
	return nil
}

func (b *binder) loadReferencedPackageSchemas(n Node) error {
	// TODO: package versions
	packageNames := stringSet{}

	if r, ok := n.(*Resource); ok {
		token := r.Syntax.Labels[1]
		packageName, _, _, _ := decomposeToken(token, r.Syntax.LabelRanges[1])
		packageNames.add(packageName)
	}

	hclsyntax.VisitAll(n.SyntaxNode(), func(node hclsyntax.Node) hcl.Diagnostics {
		call, ok := node.(*hclsyntax.FunctionCallExpr)
		if !ok {
			return nil
		}
		token, tokenRange, ok := getInvokeToken(call)
		if !ok {
			return nil
		}
		packageName, _, _, _ := decomposeToken(token, tokenRange)
		packageNames.add(packageName)
		return nil
	})

	for _, name := range packageNames.sortedValues() {
		if err := b.loadPackageSchema(name); err != nil {
			return err
		}
	}
	return nil
}

// TODO: provider versions
func (b *binder) loadPackageSchema(name string) error {
	if _, ok := b.packageSchemas[name]; ok {
		return nil
	}

	provider, err := b.host.Provider(tokens.Package(name), nil)
	if err != nil {
		return err
	}

	schemaBytes, err := provider.GetSchema(0)
	if err != nil {
		return err
	}

	var spec schema.PackageSpec
	if err := json.Unmarshal(schemaBytes, &spec); err != nil {
		return err
	}

	packageSchema, err := schema.ImportSpec(spec)
	if err != nil {
		return err
	}

	b.packageSchemas[name] = packageSchema
	return nil
}
