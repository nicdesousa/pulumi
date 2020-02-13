package pulumi

import (
	"io/ioutil"
	"mime"
	"path"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/hashicorp/hcl/v2"
	"github.com/pulumi/pulumi/pkg/resource"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var fileAssetFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "path", Type: cty.String}},
	Type:   function.StaticReturnType(assetCapsule),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		asset, err := resource.NewPathAsset(args[0].AsString())
		if err != nil {
			return cty.Value{}, err
		}
		return cty.CapsuleVal(assetCapsule, asset), nil
	},
})

var evalFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "source", Type: cty.String}},
	Type:   function.StaticReturnType(cty.DynamicPseudoType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		source := []byte("__result := " + args[0].AsString())
		script := tengo.NewScript(source)
		script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
		compiled, err := script.Run()
		if err != nil {
			return cty.Value{}, err
		}
		return convertScriptObjectToValue(compiled.Get("__result").Object())
	},
})

var funcFunc = function.New(&function.Spec{
	Params:   []function.Parameter{{Name: "source", Type: cty.String}},
	VarParam: &function.Parameter{Name: "args", Type: cty.String},
	Type:     function.StaticReturnType(cty.DynamicPseudoType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		funcBody := args[len(args)-1].AsString()

		source := []byte("__result := func() {" + funcBody + "}()")
		script := tengo.NewScript(source)
		script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

		var formalNames []string
		for _, nameVal := range args[:len(args)-1] {
			name := nameVal.AsString()
			formalNames = append(formalNames, name)
			script.Add(name, nil)
		}
		script.Add("argv", nil)

		compiled, err := script.Compile()
		if err != nil {
			return cty.Value{}, err
		}

		fn := function.New(&function.Spec{
			VarParam: &function.Parameter{Name: "args", Type: cty.DynamicPseudoType, AllowDynamicType: true},
			Type:     function.StaticReturnType(cty.DynamicPseudoType),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				scriptArgs := make([]tengo.Object, len(args))
				for i, arg := range args {
					if !arg.IsWhollyKnown() {
						return cty.UnknownVal(cty.DynamicPseudoType), nil
					}

					scriptArgs[i] = convertValueToScriptObject(arg)
				}

				clone := compiled.Clone()
				for _, name := range formalNames {
					if len(scriptArgs) == 0 {
						clone.Set(name, tengo.UndefinedValue)
						break
					}
					clone.Set(name, scriptArgs[0])
					scriptArgs = scriptArgs[1:]
				}
				clone.Set("argv", scriptArgs)

				if err := clone.Run(); err != nil {
					return cty.Value{}, err
				}
				return convertScriptObjectToValue(clone.Get("__result").Object())
			},
		})
		return cty.CapsuleVal(funcCapsule, &fn), nil
	},
})

var mimeTypeFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "filename", Type: cty.String}},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(mime.TypeByExtension(path.Ext(args[0].AsString()))), nil
	},
})

var readDirFunc = function.New(&function.Spec{
	Params: []function.Parameter{{Name: "path", Type: cty.String}},
	Type:   function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		path := args[0].AsString()
		infos, err := ioutil.ReadDir(path)
		if err != nil {
			return cty.Value{}, err
		}
		if len(infos) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}
		names := make([]cty.Value, len(infos))
		for i, info := range infos {
			names[i] = cty.StringVal(info.Name())
		}
		return cty.ListVal(names), nil
	},
})

func makeBuiltinEvalContext(ctx *programContext) *hcl.EvalContext {
	return &hcl.EvalContext{
		Functions: map[string]function.Function{
			"eval":      evalFunc,
			"func":      funcFunc,
			"fileAsset": fileAssetFunc,
			"invoke":    makeInvokeFunc(ctx),
			//"mimeType":  mimeTypeFunc,
			//"readDir": readDirFunc,
		},
	}
}
