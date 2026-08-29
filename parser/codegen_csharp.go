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

var (
	CSharpTypeMap = map[string]string{
		"int":       "int",
		"int32":     "int",
		"int64":     "long",
		"string":    "string",
		"bool":      "bool",
		"float":     "float",
		"double":    "double",
		"timestamp": "long",
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
	matches, err := findCodeTemplates(g.tplPath, false)
	if err != nil {
		slog.Error("CsharpLoaderCodeGenerator.GenCode fail", "error", err)
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

	matches, err := findCodeTemplates(g.tplPath, true)
	if err != nil {
		slog.Error("CsharpModuleCodeGenerator.GenCode fail", "error", err)
		return false
	}
	pks := sheet.GetPrimaryKeys()
	if len(pks) == 0 {
		panic(fmt.Sprintf("not found PrimaryKey when csharp GenCode, sheetName: %v", sheet.sheetName))
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
				keyType = fd.BaseType()
			} else {
				var ok bool
				keyType, ok = CSharpTypeMap[fd.BaseType()]
				if !ok {
					panic(fmt.Sprintf("invalid csharp type %v", fd.BaseType()))
				}
			}
			keys = append(keys, KeyField{Type: keyType, Name: fd.Name()})
		}

		multiKey := len(keys) > 1
		keyType := keys[0].Type
		if multiKey {
			// 多主键：使用 ValueTuple 作为字典key，如 (int, long)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k.Type)
			}
			keyType = "(" + strings.Join(parts, ", ") + ")"
		}

		m := &CodeModuleModel{
			Name:        sheet.sheetName,
			PackageName: sheet.getPackageName(),
			KeyType:     keyType,
			KeyName:     keys[0].Name,
			Keys:        keys,
			MultiKey:    multiKey,
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
