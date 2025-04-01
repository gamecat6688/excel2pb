package parser

// 加载代码生成器
type LoaderCodeGenerator interface {
	GenCode() bool
}

// 模块代码生成器
type ModuleCodeGenerator interface {
	GenCode() bool
}
