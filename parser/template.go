package parser

const (
	ProtoMessageTemplate = `
syntax = "proto3";
package {{.PackageName}};

option csharp_namespace = "{{.PackageName}}";
option go_package = "{{.GoPackagePath}}";

{{range .Imports}}import {{.ProtoPath}}
{{end}}

message {{.MessageName}}Config {
  repeated {{.MessageName}} Records = 1;
}

message {{.MessageName}} {
{{range .Fields}}  {{.ProtoType}} {{.FieldName}} = {{.FieldTag}}; // {{.Comment}}
{{end}}}
`
)

const (
	ProtoEnumTemplate = `
syntax = "proto3";
package {{.PackageName}};

option csharp_namespace = "{{.PackageName}}";
option go_package = "{{.GoPackagePath}}";


enum {{.MessageName}} {
{{range .Fields}}  {{.FieldName}} = {{.FieldTag}}; // {{.Comment}}
{{end}}}
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
	PackageName   string
	GoPackagePath string
	MessageName   string
	Imports       []ImportModel
	Fields        []FieldModel
}

type ProtoEnumModel struct {
	PackageName   string
	GoPackagePath string
	MessageName   string
	Imports       []ImportModel
	Fields        []FieldModel
}

type CodeLoaderModel struct {
	Names []string
}

// KeyField 主键字段（名称+类型），用于生成组合主键
type KeyField struct {
	Type string
	Name string
}

type CodeModuleModel struct {
	Name          string
	FullName      string
	PackageName   string
	GoPackagePath string

	// ProtoScriptPath Godot 模板使用的 protobuf GDScript 相对资源路径。
	ProtoScriptPath string

	// DataFilePath Godot 模板使用的二进制数据相对资源路径。
	DataFilePath string

	// KeyType 单主键时为主键类型；多主键时为组合key类型
	// (golang: 生成的key结构体名; csharp: ValueTuple 如 "(int, long)";
	// godot: 多主键使用 Array)
	KeyType string

	// KeyName 第一个主键的字段名，单主键模板使用
	KeyName string

	// Keys 全部主键字段，按列顺序
	Keys []KeyField

	// MultiKey 是否多主键
	MultiKey bool
}
