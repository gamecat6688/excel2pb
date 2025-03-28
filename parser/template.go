package parser

const (
	ProtoMessageTemplate = `
syntax = "proto3";
package {{.PackageName}};

option csharp_namespace = "{{.PackageName}}";
option go_package = "/{{.PackageName}}";

{{range .Imports}}import {{.ProtoPath}}
{{end}}

message {{.MessageName}}Config {
  repeated {{.MessageName}} Records = 1;
}

message {{.MessageName}} {
{{range .Fields}}  {{.ProtoType}} {{.FieldName}} = {{.FieldTag}}; // {{.Comment}}
{{end}}
}
`
)

const (
	ProtoEnumTemplate = `
syntax = "proto3";
package {{.PackageName}};

option csharp_namespace = "{{.PackageName}}";
option go_package = "/{{.PackageName}}";


enum {{.MessageName}} {
{{range .Fields}}  {{.FieldName}} = {{.FieldTag}}; // {{.Comment}}
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
	FieldTag  int32
	Comment   string
}
type ProtoMessageModel struct {
	PackageName string
	MessageName string
	Imports     []ImportModel
	Fields      []FieldModel
}

type ProtoEnumModel struct {
	PackageName string
	MessageName string
	Imports     []ImportModel
	Fields      []FieldModel
}
