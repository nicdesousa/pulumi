package pulumi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2/hcldec"
	pschema "github.com/pulumi/pulumi/pkg/codegen/schema"
	"github.com/zclconf/go-cty/cty"
)

type resourceSchema struct {
	pulumi *pschema.Resource
	spec   hcldec.ObjectSpec
}

type functionSchema struct {
	pulumi     *pschema.Function
	argsType   cty.Type
	returnType cty.Type
}

type packageSchema struct {
	pulumi    *pschema.Package
	resources map[string]*resourceSchema
	functions map[string]*functionSchema
}

func decomposeToken(tok string) (string, string, string) {
	components := strings.Split(tok, ":")
	return components[0], components[1], components[2]
}

func canonicalizeToken(tok string, pkg *pschema.Package) string {
	_, _, member := decomposeToken(tok)
	return fmt.Sprintf("%s:%s:%s", pkg.Name, pkg.TokenToModule(tok), member)
}

func loadSchema(pkgName string) (*packageSchema, error) {
	f, err := os.Open("./" + pkgName + ".json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var spec pschema.PackageSpec
	if err := json.NewDecoder(f).Decode(&spec); err != nil {
		return nil, err
	}

	pkg, err := pschema.ImportSpec(spec)
	if err != nil {
		return nil, err
	}

	resources := map[string]*resourceSchema{}
	for _, r := range pkg.Resources {
		resources[canonicalizeToken(r.Token, pkg)] = &resourceSchema{
			pulumi: r,
			spec:   makeObjectSpec(r.InputProperties),
		}
	}

	functions := map[string]*functionSchema{}
	for _, f := range pkg.Functions {
		var argsType, returnType cty.Type
		if f.Inputs != nil {
			argsType = makeType(f.Inputs)
		}
		if f.Outputs != nil {
			returnType = makeType(f.Outputs)
		}

		functions[canonicalizeToken(f.Token, pkg)] = &functionSchema{
			pulumi:     f,
			argsType:   argsType,
			returnType: returnType,
		}
	}

	return &packageSchema{pulumi: pkg, resources: resources, functions: functions}, nil
}
