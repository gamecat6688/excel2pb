package parser

import (
	"excel2pb/config"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

var GodotTypeMap = map[string]string{
	"int":       "int",
	"int32":     "int",
	"int64":     "int",
	"string":    "String",
	"bool":      "bool",
	"float":     "float",
	"double":    "float",
	"timestamp": "int",
	"i18n":      "String",
}

// GodotLoaderCodeGenerator 生成汇总加载入口。
type GodotLoaderCodeGenerator struct {
	tplPath, outPath string
}

func NewGodotLoaderCode(tplPath, outPath string) *GodotLoaderCodeGenerator {
	return &GodotLoaderCodeGenerator{tplPath: tplPath, outPath: outPath}
}

func (g *GodotLoaderCodeGenerator) GenCode(root *Parser) bool {
	matches, err := findCodeTemplates(g.tplPath, false)
	if err != nil {
		slog.Error("GodotLoaderCodeGenerator.GenCode fail", "error", err)
		return false
	}

	if err := os.MkdirAll(g.outPath, os.ModePerm); err != nil {
		slog.Error("create Godot code output directory fail", "error", err)
		return false
	}

	model := &CodeLoaderModel{}
	for _, sheet := range root.sheets {
		if sheet.HasData() {
			model.Names = append(model.Names, sheet.sheetName)
		}
	}
	slices.Sort(model.Names)

	ok := true
	for _, filename := range matches {
		if !executeGodotTemplate(filename, filepath.Join(g.outPath, filepath.Base(filename)), model) {
			ok = false
		}
	}
	return ok
}

// GodotModuleCodeGenerator 为每张数据表生成查询模型。
type GodotModuleCodeGenerator struct {
	tplPath, outPath string
}

func NewGodotModuleCode(tplPath, outPath string) *GodotModuleCodeGenerator {
	return &GodotModuleCodeGenerator{tplPath: tplPath, outPath: outPath}
}

func (g *GodotModuleCodeGenerator) GenCode(root *Parser, sheet *SheetParser) bool {
	if !sheet.HasData() {
		return false
	}

	matches, err := findCodeTemplates(g.tplPath, true)
	if err != nil {
		slog.Error("GodotModuleCodeGenerator.GenCode fail", "error", err)
		return false
	}

	pks := sheet.GetPrimaryKeys()
	if len(pks) == 0 {
		panic(fmt.Sprintf("not found PrimaryKey when godot GenCode, sheetName: %v", sheet.sheetName))
	}

	keys := make([]KeyField, 0, len(pks))
	for _, field := range pks {
		keyType := "int"
		if !field.IsCustomEnum(root) {
			var exists bool
			keyType, exists = GodotTypeMap[field.BaseType()]
			if !exists {
				panic(fmt.Sprintf("invalid godot type %v", field.BaseType()))
			}
		}
		keys = append(keys, KeyField{Type: keyType, Name: field.Name()})
	}

	multiKey := len(keys) > 1
	keyType := keys[0].Type
	if multiKey {
		keyType = "Array"
	}

	cfg := configForSheet(sheet)
	model := &CodeModuleModel{
		Name:            sheet.sheetName,
		KeyType:         keyType,
		KeyName:         keys[0].Name,
		Keys:            keys,
		MultiKey:        multiKey,
		ProtoScriptPath: relativeGodotPath(g.outPath, filepath.Join(cfg.PbPath, sheet.sheetName+".proto.gd"), true),
		DataFilePath:    relativeGodotPath(g.outPath, filepath.Join(cfg.DataPath, sheet.sheetName+cfg.DataExt), false),
	}

	if err := os.MkdirAll(g.outPath, os.ModePerm); err != nil {
		slog.Error("create Godot code output directory fail", "error", err)
		return false
	}

	ok := true
	for _, filename := range matches {
		baseName := filepath.Base(filename)
		fixedName := ""
		switch {
		case strings.Contains(baseName, "{name}"):
			fixedName = strings.ReplaceAll(baseName, "{name}", strings.ToLower(sheet.sheetName))
		case strings.Contains(baseName, "{Name}"):
			fixedName = strings.ReplaceAll(baseName, "{Name}", sheet.sheetName)
		default:
			panic("invalid template name")
		}

		if !executeGodotTemplate(filename, filepath.Join(g.outPath, fixedName), model) {
			ok = false
		}
	}
	return ok
}

func configForSheet(sheet *SheetParser) config.OutConfig {
	return config.Cfg.Outs[FilterFullName[sheet.filter]]
}

func relativeGodotPath(fromDir, target string, preload bool) string {
	rel, err := filepath.Rel(fromDir, target)
	if err != nil {
		rel = target
	}
	rel = filepath.ToSlash(rel)
	if preload && !strings.HasPrefix(rel, ".") && !strings.HasPrefix(rel, "/") {
		rel = "./" + rel
	}
	return rel
}

func executeGodotTemplate(filename, outputPath string, model any) bool {
	tmpl, err := template.ParseFiles(filename)
	if err != nil {
		slog.Error("parse Godot code template fail", "file", filename, "error", err)
		return false
	}

	f, err := os.Create(outputPath)
	if err != nil {
		slog.Error("create Godot code file fail", "file", outputPath, "error", err)
		return false
	}

	if err := tmpl.Execute(f, model); err != nil {
		_ = f.Close()
		slog.Error("execute Godot code template fail", "file", filename, "error", err)
		return false
	}
	if err := f.Close(); err != nil {
		slog.Error("close Godot code file fail", "file", outputPath, "error", err)
		return false
	}
	return true
}
