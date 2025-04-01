package parser

// 加载
type CsharpLoaderCodeGenerator struct {
}

func NewCsharpLoaderCode(tplPath, outPath string) *CsharpLoaderCodeGenerator {
	return &CsharpLoaderCodeGenerator{}
}

func (g *CsharpLoaderCodeGenerator) ImportTemplate() bool {
	return true
}

func (g *CsharpLoaderCodeGenerator) GenCode(root *Parser) bool {

	return true
}

// 模块
type CsharpModuleCodeGenerator struct {
}

func NewCsharpModuleCode(tplPath, outPath string) *CsharpModuleCodeGenerator {
	return &CsharpModuleCodeGenerator{}
}

func (g *CsharpModuleCodeGenerator) ImportTemplate() bool {
	return true
}

func (g *CsharpModuleCodeGenerator) GenCode(sheet *SheetParser) bool {
	return true
}
