package syntax

import (
	"io"
	"io/ioutil"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type File struct {
	Name  string
	Body  *hclsyntax.Body
	Bytes []byte
}

type Parser struct {
	Files []*File

	diagnostics hcl.Diagnostics
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFile(r io.Reader, filename string) error {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	hclFile, diags := hclsyntax.ParseConfig(src, filename, hcl.Pos{})
	p.Files = append(p.Files, &File{
		Name:  filename,
		Body:  hclFile.Body.(*hclsyntax.Body),
		Bytes: hclFile.Bytes,
	})
	p.diagnostics = append(p.diagnostics, diags...)
	return nil
}

func (p *Parser) Diagnostics() hcl.Diagnostics {
	return p.diagnostics
}

func (p *Parser) NewDiagnosticWriter(w io.Writer, width uint, color bool) hcl.DiagnosticWriter {
	return NewDiagnosticWriter(w, p.Files, width, color)
}

func NewDiagnosticWriter(w io.Writer, files []*File, width uint, color bool) hcl.DiagnosticWriter {
	fileMap := map[string]*hcl.File{}
	for _, f := range files {
		fileMap[f.Name] = &hcl.File{Body: f.Body, Bytes: f.Bytes}
	}
	return hcl.NewDiagnosticTextWriter(w, fileMap, width, color)
}
