package codegen

// 枚举代码生成器
type EnumCodeGenerator interface {
	ImportTemplate() bool
	GenCode() bool
}

// 数据表代码生成器
type TableCodeGenerator interface {
	ImportTemplate() bool
	GenCode() bool
}
