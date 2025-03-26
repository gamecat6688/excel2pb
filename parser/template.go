package parser

const (
	ProtoTemplate = `
syntax = "proto3";
package {{.PackageName}};

option csharp_namespace = "{{.PackageName}}";
option go_package = "/{{.PackageName}}";

{{range .Imports}}
import {{.ProtoPath}}
{{end}}

message {{.SheetName}}Config {
  repeated {{.SheetName}} Data = 1;
}

message {{.SheetName}} {
{{range .Fields}}  {{.ProtoType}} {{.FieldName}} = {{.FieldTag}}; // {{.Comment}}
{{end}}
}
`
)

type ImportModel struct {
	ProtoPath string
}

type FieldModel struct {
	ProtoType string
	FieldName string
	FieldTag  int
	Comment   string
}
type ProtoModel struct {
	PackageName string
	SheetName   string
	Imports     []ImportModel
	Fields      []FieldModel
}
