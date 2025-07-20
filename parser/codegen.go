package parser

// 加载代码生成器
type LoaderCodeGenerator interface {
	GenCode(root *Parser, sheet *SheetParser) bool
}

// 模块代码生成器
type ModuleCodeGenerator interface {
	GenCode(root *Parser, sheet *SheetParser) bool
}
