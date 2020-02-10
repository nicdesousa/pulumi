package pulumi

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	pschema "github.com/pulumi/pulumi/pkg/codegen/schema"
)

func makePropertySpec(p *pschema.Property) hcldec.Spec {
	switch t := p.Type.(type) {
	case *pschema.ArrayType:
		if et, ok := t.ElementType.(*pschema.ObjectType); ok {
			spec := &hcldec.BlockListSpec{
				TypeName: p.Name,
				Nested:   makeObjectSpec(et.Properties),
			}
			if p.IsRequired {
				spec.MinItems = 1
			}
			return spec
		}
	case *pschema.MapType:
		if et, ok := t.ElementType.(*pschema.ObjectType); ok {
			return &hcldec.BlockMapSpec{
				TypeName:   p.Name,
				LabelNames: []string{"key"},
				Nested:     makeObjectSpec(et.Properties),
			}
		}
	case *pschema.ObjectType:
		return &hcldec.BlockSpec{
			TypeName: p.Name,
			Nested:   makeObjectSpec(t.Properties),
			Required: p.IsRequired,
		}
	}

	return &hcldec.AttrSpec{
		Name:     p.Name,
		Type:     makeType(p.Type),
		Required: p.IsRequired,
	}
}

func makeObjectSpec(properties []*pschema.Property) hcldec.ObjectSpec {
	spec := hcldec.ObjectSpec{}
	for _, p := range properties {
		spec[p.Name] = makePropertySpec(p)
	}
	return spec
}
