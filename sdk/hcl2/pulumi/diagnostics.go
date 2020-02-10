package pulumi

import "github.com/hashicorp/hcl/v2"

func unknownResource(name string, subject hcl.Range) *hcl.Diagnostic {
	message := "unknown resource " + name
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  message,
		Detail:   message,
		Subject:  &subject,
	}
}

func invalidRange(subject hcl.Range) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "invalid type for range expression",
		Detail:   "range expressions must be numbers, lists, or maps",
		Subject:  &subject,
	}
}
