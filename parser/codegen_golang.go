package parser

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

var GolangTypeMap = map[string]string{
	"int32":     "int32",
	"int64":     "int64",
	"string":    "string",
	"bool":      "bool",
	"float":     "float32",
	"double":    "float64",
	"timestamp": "int64",
}

// 加载
type GolangLoaderCodeGenerator struct {
	tplPath, outPath string
}

func NewGolangLoaderCode(tplPath, outPath string) *GolangLoaderCodeGenerator {
	return &GolangLoaderCodeGenerator{
		tplPath: tplPath,
		outPath: outPath,
	}
}

func (g *GolangLoaderCodeGenerator) GenCode(root *Parser) bool {
	matches, err := findCodeTemplates(g.tplPath, false)
	if err != nil {
		slog.Error("GolangLoaderCodeGenerator.GenCode fail", "error", err)
		return false
	}
	ok := true

	for _, filename := range matches {
		// 解析模板
		tmpl, err := template.ParseFiles(filename)
		if err != nil {
			slog.Error("parseFromFile proto message template fail", "error", err)
			ok = false
			continue
		}

		// 创建proto文件
		outPath := g.outPath
		if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
			slog.Error("create code output directory fail", "path", outPath, "error", err)
			ok = false
			continue
		}

		f, err := os.Create(filepath.Join(outPath, filepath.Base(filename)))
		if err != nil {
			slog.Error("create code file fail", "error", err)
			ok = false
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
			_ = f.Close()
			slog.Error("tmpl.Execute fail", "error", err)
			ok = false
			continue
		}
		if err := f.Close(); err != nil {
			slog.Error("close code file fail", "error", err)
			ok = false
		}
	}

	return ok
}

// 模块
type GolangModuleCodeGenerator struct {
	tplPath, outPath string
}

func NewGolangModuleCode(tplPath, outPath string) *GolangModuleCodeGenerator {
	return &GolangModuleCodeGenerator{
		tplPath: tplPath,
		outPath: outPath,
	}
}

func (g *GolangModuleCodeGenerator) ImportTemplate() bool {
	return true
}

func (g *GolangModuleCodeGenerator) GenCode(root *Parser, sheet *SheetParser) bool {
	if !sheet.HasData() {
		// 没有数据，不生成代码
		return false
	}

	matches, err := findCodeTemplates(g.tplPath, true)
	if err != nil {
		slog.Error("GolangModuleCodeGenerator.GenCode fail", "error", err)
		return false
	}
	pks := sheet.GetPrimaryKeys()
	if len(pks) == 0 {
		panic(fmt.Sprintf("not found PrimaryKey when golang GenCode, sheetName: %v", sheet.sheetName))
	}
	ok := true

	for _, filename := range matches {
		// 解析模板
		tmpl, err := template.ParseFiles(filename)
		if err != nil {
			slog.Error("parseFromFile proto message template fail", "error", err)
			ok = false
			continue
		}

		// 创建proto文件
		outPath := g.outPath
		if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
			slog.Error("create code output directory fail", "path", outPath, "error", err)
			ok = false
			continue
		}

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
			ok = false
			continue
		}

		// 执行模板,输出文件
		keys := make([]KeyField, 0, len(pks))
		for _, fd := range pks {
			var keyType string
			if fd.IsCustomEnum(root) {
				keyType = fmt.Sprintf("%v.%v", sheet.getPackageName(), fd.BaseType())
			} else {
				var exists bool
				keyType, exists = GolangTypeMap[fd.BaseType()]
				if !exists {
					panic(fmt.Sprintf("invalid golang primary key type %v", fd.BaseType()))
				}
			}
			keys = append(keys, KeyField{Type: keyType, Name: fd.Name()})
		}

		multiKey := len(keys) > 1
		keyType := keys[0].Type
		if multiKey {
			// 多主键：生成组合key结构体，类型名为 <表名>Key
			keyType = sheet.sheetName + "Key"
		}

		m := &CodeModuleModel{
			Name:          sheet.sheetName,
			FullName:      fmt.Sprintf("%v.%v", sheet.getPackageName(), sheet.sheetName),
			PackageName:   sheet.getPackageName(),
			GoPackagePath: sheet.getGoPackagePath(),
			KeyType:       keyType,
			KeyName:       keys[0].Name,
			Keys:          keys,
			MultiKey:      multiKey,
		}
		err = tmpl.Execute(f, m)
		if err != nil {
			_ = f.Close()
			slog.Error("tmpl.Execute fail", "error", err)
			ok = false
			continue
		}
		if err := f.Close(); err != nil {
			slog.Error("close code file fail", "error", err)
			ok = false
		}
	}

	return ok
}
