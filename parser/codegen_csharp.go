package parser

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

var (
	CSharpTypeMap = map[string]string{
		"int":    "int",
		"int32":  "int",
		"int64":  "long",
		"string": "string",
		"bool":   "bool",
		"float":  "float",
		"double": "double",
	}
)

// 加载
type CsharpLoaderCodeGenerator struct {
	tplPath, outPath string
}

func NewCsharpLoaderCode(tplPath, outPath string) *CsharpLoaderCodeGenerator {
	return &CsharpLoaderCodeGenerator{
		tplPath: tplPath,
		outPath: outPath,
	}
}

func (g *CsharpLoaderCodeGenerator) GenCode(root *Parser) bool {
	matches, err := filepath.Glob(g.tplPath + "*.*")
	if err != nil {
		slog.Error("GolangLoaderCodeGenerator.GenCode fail", "error", err)
	}

	for _, filename := range matches {
		if strings.Index(filename, "{") >= 0 {
			continue
		}

		// 解析模板
		tmpl, err := template.ParseFiles(filename)
		if err != nil {
			slog.Error("parseFromFile proto message template fail", "error", err)
			continue
		}

		// 创建proto文件
		outPath := g.outPath
		os.MkdirAll(outPath, os.ModePerm)

		f, err := os.Create(filepath.Join(outPath, filepath.Base(filename)))
		if err != nil {
			slog.Error("create code file fail", "error", err)
			continue
		}

		// 执行模板,输出文件
		m := &CodeLoaderModel{}
		for _, v := range root.sheets {
			if !v.HasData() {
				continue
			}

			m.Names = append(m.Names, v.sheetName)
		}
		slices.Sort(m.Names)
		err = tmpl.Execute(f, m)
		if err != nil {
			slog.Error("tmpl.Execute fail", "error", err)
			continue
		}
	}

	return true
}

// 模块
type CsharpModuleCodeGenerator struct {
	tplPath, outPath string
}

func NewCsharpModuleCode(tplPath, outPath string) *CsharpModuleCodeGenerator {
	return &CsharpModuleCodeGenerator{
		tplPath: tplPath,
		outPath: outPath,
	}
}

func (g *CsharpModuleCodeGenerator) GenCode(root *Parser, sheet *SheetParser) bool {
	if !sheet.HasData() {
		// 没有数据，不生成代码
		return false
	}

	matches, err := filepath.Glob(g.tplPath + "*.*")
	if err != nil {
		slog.Error("GolangModuleCodeGenerator.GenCode fail", "error", err)
	}

	for _, filename := range matches {
		if strings.Index(filename, "{") == -1 {
			continue
		}

		// 解析模板
		tmpl, err := template.ParseFiles(filename)
		if err != nil {
			slog.Error("parseFromFile proto message template fail", "error", err)
			continue
		}

		// 创建proto文件
		outPath := g.outPath
		os.MkdirAll(outPath, os.ModePerm)

		var fixedName string
		baseName := filepath.Base(filename)
		if strings.Index(baseName, `{name}`) != -1 {
			fixedName = strings.Replace(baseName, `{name}`, strings.ToLower(sheet.sheetName), -1)
		} else if strings.Index(baseName, `{Name}`) != -1 {
			fixedName = strings.Replace(baseName, `{Name}`, sheet.sheetName, -1)
		} else {
			panic("invalid template name")
		}

		f, err := os.Create(filepath.Join(outPath, fixedName))
		if err != nil {
			slog.Error("create code file fail", "error", err)
			continue
		}

		// 执行模板,输出文件
		var keyType, keyName string
		if len(sheet.GetPrimaryKeys()) > 0 {
			// TODO 取第一个主键的类型，后续可以扩展
			fd := sheet.GetPrimaryKeys()[0]
			keyType = fd.BaseType()
			keyName = fd.Name()
		}
		m := &CodeModuleModel{
			Name:    sheet.sheetName,
			KeyType: CSharpTypeMap[keyType],
			KeyName: keyName,
		}
		err = tmpl.Execute(f, m)
		if err != nil {
			slog.Error("tmpl.Execute fail", "error", err)
			continue
		}
	}

	return true
}
