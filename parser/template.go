package parser

const (
	ProtoTemplate = `
syntax = "proto3";
package {{.PackageName}};

option csharp_namespace = "{{.PackageName}}";
option go_package = "/{{.PackageName}}";


message {{.SheetName}} {
{{range .Fields}}  {{.ProtoType}} {{.FieldName}} = {{.FieldTag}}; // {{.Comment}}
{{end}}
}
`
)

type FieldModel struct {
	ProtoType string
	FieldName string
	FieldTag  int
	Comment   string
}
type ProtoModel struct {
	PackageName string
	SheetName   string
	Fields      []FieldModel
}
