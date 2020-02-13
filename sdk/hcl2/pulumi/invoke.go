package pulumi

import (
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/pulumi/pulumi/pkg/resource/plugin"
	pulumirpc "github.com/pulumi/pulumi/sdk/proto/go"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"go.uber.org/multierr"
)

var invokeOptionsType = cty.Object(map[string]cty.Type{
	"parent":   cty.String,
	"provider": cty.String,
})

func checkAssignability(dstType, srcType cty.Type, path cty.Path) error {
	if dstType == cty.DynamicPseudoType || srcType == cty.DynamicPseudoType {
		// Anything is assignable to or from the dynamic type.
		return nil
	}

	if dstType.IsObjectType() && srcType.IsObjectType() {
		var errs []error

		// Make space for our last path element.
		path = append(path, nil)

		// This is a relaxed version of the usual type equality used by TestConformance.
		dstAttributeTypes := dstType.AttributeTypes()
		for name, srcAttributeType := range srcType.AttributeTypes() {
			dstAttributeType, ok := dstAttributeTypes[name]
			if !ok {
				errs = append(errs, path.NewErrorf("unsupported attribute %q", name))
			} else {
				if err := checkAssignability(dstAttributeType, srcAttributeType, append(path, cty.GetAttrStep{Name: name})); err != nil {
					errs = append(errs, err)
				}
			}
		}

		if len(errs) != 0 {
			return multierr.Combine(errs...)
		}
		return nil
	}

	errs := srcType.TestConformance(dstType)
	if len(errs) != 0 {
		return multierr.Combine(errs...)
	}
	return nil
}

func makeInvokeFunc(ctx *programContext) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{Name: "function", Type: cty.String},
			{Name: "args", Type: cty.DynamicPseudoType},
		},
		VarParam: &function.Parameter{Name: "options", Type: invokeOptionsType},
		Type: func(args []cty.Value) (cty.Type, error) {
			if !args[0].IsKnown() {
				return cty.DynamicPseudoType, nil
			}

			token := args[0].AsString()

			pkgName, _, _ := decomposeToken(token)
			pkgSchema, err := ctx.ensureSchema(pkgName)
			if err != nil {
				return cty.Type{}, err
			}
			//functionSchema, ok := pkgSchema.functions[token]
			_, ok := pkgSchema.functions[token]
			if !ok {
				return cty.Type{}, errors.Errorf("unknown function %s", token)
			}

			return cty.DynamicPseudoType, nil
			//return functionSchema.returnType, nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			token := args[0].AsString()

			pkgName, _, _ := decomposeToken(token)
			pkgSchema, err := ctx.ensureSchema(pkgName)
			if err != nil {
				return cty.Value{}, err
			}
			functionSchema, ok := pkgSchema.functions[token]
			if !ok {
				return cty.Value{}, errors.Errorf("unknown function %s", token)
			}

			if err := checkAssignability(functionSchema.argsType, args[1].Type(), nil); err != nil {
				return cty.Value{}, err
			}

			invokeArgs := marshalValue(args[1]).ObjectValue()

			rpcArgs, err := plugin.MarshalProperties(invokeArgs, plugin.MarshalOptions{KeepUnknowns: false})
			if err != nil {
				return cty.Value{}, errors.Wrap(err, "marshaling arguments")
			}

			resp, err := ctx.monitor.Invoke(ctx.cancel, &pulumirpc.InvokeRequest{
				Tok:  functionSchema.pulumi.Token,
				Args: rpcArgs,
				// Provider:
			})
			if err != nil {
				return cty.Value{}, err
			}

			if len(resp.Failures) > 0 {
				errs := make([]error, len(resp.Failures))
				for i, f := range resp.Failures {
					errs[i] = errors.Errorf("failed to invoke %s: %s (%s)", token, f.Reason, f.Property)
				}
				return cty.Value{}, multierr.Combine(errs...)
			}

			retVal, err := plugin.UnmarshalProperties(resp.Return, plugin.MarshalOptions{KeepSecrets: true})
			if err != nil {
				return cty.Value{}, err
			}

			if functionSchema.pulumi.Outputs != nil {
				for _, prop := range functionSchema.pulumi.Outputs.Properties {
					k := resource.PropertyKey(prop.Name)
					if _, has := retVal[k]; !has {
						retVal[k] = resource.PropertyValue{}
					}
				}
			}

			return unmarshalValue(resource.NewObjectProperty(retVal)), nil
		},
	})
}
