package pulumi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2/hcldec"
	pschema "github.com/pulumi/pulumi/pkg/codegen/schema"
)

type resourceSchema struct {
	pulumi *pschema.Resource
	spec   hcldec.ObjectSpec
}

type packageSchema struct {
	pulumi    *pschema.Package
	resources map[string]*resourceSchema
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

	return &packageSchema{pulumi: pkg, resources: resources}, nil
}
