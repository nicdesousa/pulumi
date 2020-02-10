package pulumi

import (
	"io/ioutil"
	"mime"
	"path"

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

var builtinFunctions = map[string]function.Function{
	"fileAsset": fileAssetFunc,
	"mimeType":  mimeTypeFunc,
	"readDir":   readDirFunc,
}
