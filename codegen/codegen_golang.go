package codegen

// 枚举代码生成器
type GolangEnumCodeGenerator struct {
}

func (g *GolangEnumCodeGenerator) ImportTemplate() bool {
	return true
}

func (g *GolangEnumCodeGenerator) GenCode() bool {
	return true
}

// 数据表代码生成器
type GolangTableCodeGenerator struct {
}

func (g *GolangTableCodeGenerator) ImportTemplate() bool {
	return true
}

func (g *GolangTableCodeGenerator) GenCode() bool {
	return true
}
