package parser

import (
	"excel2pb/config"
	"github.com/xuri/excelize/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func configureTestExports(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	previous := config.Cfg
	config.Cfg = &config.Config{
		TimeZone:        "+08:00",
		EnableI18n:      true,
		ExcelDir:        filepath.Join(dir, "xls"),
		ProtoImportPath: "",
		Outs: map[string]config.OutConfig{
			"Client": {ProtoPath: filepath.Join(dir, "proto", "client"), DataPath: filepath.Join(dir, "data", "client"), DataExt: ".bytes", PbPath: filepath.Join(dir, "pb", "client"), PackageName: "test", CodeLanguage: "csharp"},
			"Server": {ProtoPath: filepath.Join(dir, "proto", "server"), DataPath: filepath.Join(dir, "data", "server"), DataExt: ".data", PbPath: filepath.Join(dir, "pb", "server"), PackageName: "test", GoPackagePath: "example.com/server/test", GoModulePath: "example.com/server", CodeLanguage: "golang"},
		},
		TplCodePaths: map[string]string{
			"golang": filepath.Join(dir, "templates", "golang") + string(filepath.Separator),
			"csharp": filepath.Join(dir, "templates", "csharp") + string(filepath.Separator),
			"godot":  filepath.Join("..", "assets", "template", "godot") + string(filepath.Separator),
		},
		CodeOutPaths: map[string]string{
			"golang": filepath.Join(dir, "code", "golang"),
			"csharp": filepath.Join(dir, "code", "csharp"),
			"godot":  filepath.Join(dir, "code", "godot"),
		},
	}
	t.Cleanup(func() { config.Cfg = previous })
	return dir
}

func TestGodotCodeGeneration(t *testing.T) {
	dir := configureTestExports(t)
	client := config.Cfg.Outs["Client"]
	client.CodeLanguage = "godot"
	config.Cfg.Outs["Client"] = client
	writeTestFile(t, filepath.Join(config.Cfg.TplCodePaths["golang"], "loader.go"), "{{range .Names}}{{.}} {{end}}")
	writeTestFile(t, filepath.Join(config.Cfg.TplCodePaths["golang"], "data_{name}.go"), "{{.Name}} {{.KeyType}}")

	root := NewParser()
	attr := makeSheet("Attr", [][]string{
		{"RaceID", "Type", "Value"},
		{"pk int32", "pk string", "int64"},
		{"cs", "cs", "cs"},
		{"race", "type", "value"},
		{"1", "Attack", "100"},
	})
	root.sheets[attr.sheetName] = attr

	root.exportCode()

	modulePath := filepath.Join(config.Cfg.CodeOutPaths["godot"], "AttrModel.gd")
	module, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatalf("read generated Godot module: %v", err)
	}
	output := string(module)
	for _, want := range []string{
		"class_name AttrModel",
		"const Proto = preload(\"../../pb/client/Attr.proto.gd\")",
		"const DEFAULT_DATA_FILE := \"../../data/client/Attr.bytes\"",
		"rows[[row.RaceID, row.Type]] = row",
		"static func make_key(RaceID: int, Type: String) -> Array:",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Godot module missing %q:\n%s", want, output)
		}
	}

	loader, err := os.ReadFile(filepath.Join(config.Cfg.CodeOutPaths["godot"], "game_data.gd"))
	if err != nil {
		t.Fatalf("read generated Godot loader: %v", err)
	}
	if !strings.Contains(string(loader), "if not _load_model(\"Attr\", AttrModel.new()):") {
		t.Fatalf("unexpected Godot loader:\n%s", loader)
	}

	if arg, ok := protobufOutArg("godot", filepath.Join(dir, "pb", "client")); !ok || !strings.HasPrefix(arg, "--gdscript_out=") {
		t.Fatalf("unexpected Godot protoc argument: %q, %v", arg, ok)
	}
}

func TestExportPipelineWithoutProtoc(t *testing.T) {
	dir := configureTestExports(t)
	for path, content := range map[string]string{
		filepath.Join(dir, "templates", "golang", "loader.go"):      "{{range .Names}}{{.}} {{end}}",
		filepath.Join(dir, "templates", "golang", "data_{name}.go"): "{{.Name}} {{.KeyType}}",
		filepath.Join(dir, "templates", "csharp", "loader.cs"):      "{{range .Names}}{{.}} {{end}}",
		filepath.Join(dir, "templates", "csharp", "data_{Name}.cs"): "{{.Name}} {{.KeyType}}",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	root := NewParser()
	item := makeSheet("Item", [][]string{
		{"ID", "Name", "Quality", "CreatedAt", "Description"},
		{"pk int32", "string", "Quality", "timestamp", "i18n"},
		{"cs", "cs", "cs", "cs", "cs"},
		{"id", "name", "quality", "created", "description"},
		{"1", "Sword", "Rare", "2025-03-01 08:00:05", "A sword"},
	})
	quality := &EnumParser{sheetName: "Quality_Enum", name: "Quality", enums: []EnumInfo{{Name: "None", Value: "0"}, {Name: "Rare", Value: "2"}}}
	root.sheets[item.sheetName] = item
	root.enums[quality.name] = quality

	root.exportProto()
	root.exportData()
	root.exportCode()

	for _, filter := range AllFilters {
		out := config.Cfg.Outs[FilterFullName[filter]]
		protoData, err := os.ReadFile(filepath.Join(out.ProtoPath, "Item.proto"))
		if err != nil {
			t.Fatalf("read %s Item proto: %v", filter, err)
		}
		if !strings.Contains(string(protoData), "import \"Quality.proto\";") || !strings.Contains(string(protoData), "int64 CreatedAt") {
			t.Fatalf("unexpected Item proto for %s:\n%s", filter, protoData)
		}
		data, err := os.ReadFile(filepath.Join(out.DataPath, "Item"+out.DataExt))
		if err != nil || len(data) == 0 {
			t.Fatalf("missing binary data for %s: len=%d err=%v", filter, len(data), err)
		}
	}

	if out, err := os.ReadFile(filepath.Join(config.Cfg.CodeOutPaths["golang"], "data_item.go")); err != nil || string(out) != "Item int32" {
		t.Fatalf("unexpected generated Go module: %q, %v", out, err)
	}
	if out, err := os.ReadFile(filepath.Join(config.Cfg.CodeOutPaths["csharp"], "data_Item.cs")); err != nil || string(out) != "Item int" {
		t.Fatalf("unexpected generated C# module: %q, %v", out, err)
	}
}

func TestNestedMessageExportHonorsTargetFieldFilters(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	child := makeSheet("Child", [][]string{
		{"ClientValue", "ServerValue"},
		{"string", "string"},
		{"c", "s"},
		{"client", "server"},
	})
	parent := makeSheet("Parent", [][]string{
		{"ID", "Data"},
		{"pk int32", "Child"},
		{"cs", "cs"},
		{"id", "data"},
		{"1", "client-value|server-value"},
	})
	root.sheets[child.sheetName] = child
	root.sheets[parent.sheetName] = parent

	root.checks()
	root.exportProto()
	root.exportData()
	for _, filter := range AllFilters {
		cfg := config.Cfg.Outs[FilterFullName[filter]]
		data, err := os.ReadFile(filepath.Join(cfg.DataPath, "Parent"+cfg.DataExt))
		if err != nil || len(data) == 0 {
			t.Fatalf("filtered nested data for %q was not exported: len=%d err=%v", filter, len(data), err)
		}
	}
}

func TestParserMergeI18nAndLookup(t *testing.T) {
	configureTestExports(t)
	if err := os.MkdirAll(config.Cfg.ExcelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	root := NewParser()
	item := makeSheet("Item", [][]string{
		{"ID", "Description"},
		{"pk int32", "i18n"},
		{"cs", "cs"},
		{"", ""},
		{"2", "second"},
		{"1", "first"},
	})
	root.sheets[item.sheetName] = item
	root.MergeI18n()

	i18n := root.getSheetParser(I18nSheetName)
	if i18n == nil || len(i18n.dataRows) != 2 || i18n.dataRows[0][0] != "Item_Description_1" {
		t.Fatalf("unexpected merged i18n rows: %#v", i18n)
	}
	if _, err := os.Stat(root.GetI18nFilePath()); err != nil {
		t.Fatalf("I18N workbook was not written: %v", err)
	}
	if !root.hasSheetParser("Item") || root.getSheetParser("missing") != nil || root.hasEnumParser("missing") || root.getEnumParser("missing") != nil {
		t.Fatal("parser lookup helpers returned unexpected values")
	}
}

func TestParseExcelsDoesNotExportI18nWorkbookWhenDisabled(t *testing.T) {
	configureTestExports(t)
	config.Cfg.EnableI18n = false
	writeRowsWorkbook(t, filepath.Join(config.Cfg.ExcelDir, "Item.xlsx"), "Item", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"},
	})
	writeRowsWorkbook(t, filepath.Join(config.Cfg.ExcelDir, "I18N.xlsx"), I18nSheetName, [][]string{
		{"ID", "Cn"}, {"pk string", "string"}, {"cs", "cs"}, {"", ""}, {"Item_Name_1", "名称"},
	})

	root := NewParser()
	root.ParseExcels()
	if !root.hasSheetParser("Item") {
		t.Fatal("source workbook was not parsed")
	}
	if root.hasSheetParser(I18nSheetName) {
		t.Fatal("I18N.xlsx was parsed as a normal source workbook while i18n was disabled")
	}
}

func TestMergeI18nValidatesBeforeWritingWorkbook(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	item := makeSheet("Item", [][]string{
		{"Description"},
		{"i18n"},
		{"cs"},
		{"description"},
		{"first"},
		{"second"},
	})
	root.sheets[item.sheetName] = item

	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "must define a primary key") {
			t.Fatalf("unexpected pre-merge validation result: %v", recovered)
		}
		if _, err := os.Stat(root.GetI18nFilePath()); !os.IsNotExist(err) {
			t.Fatalf("I18N workbook was written before validation: %v", err)
		}
	}()
	root.MergeI18n()
}

func writeRowsWorkbook(t *testing.T, path, sheet string, rows [][]string) {
	t.Helper()
	book := excelize.NewFile()
	if err := book.SetSheetName("Sheet1", sheet); err != nil {
		t.Fatal(err)
	}
	for rowIndex, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, rowIndex+1)
		if err != nil {
			t.Fatal(err)
		}
		if err := book.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := book.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	if err := book.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProtoFilesForFilterExcludesStaleFiles(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	root.sheets["Item"] = makeSheet("Item", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"},
	})
	root.enums["Quality"] = &EnumParser{name: "Quality"}

	files := root.protoFilesForFilter(ClientFlag)
	if len(files) != 2 || filepath.Base(files[0]) != "Item.proto" || filepath.Base(files[1]) != "Quality.proto" {
		t.Fatalf("unexpected current proto file list: %#v", files)
	}
}

func TestI18nSortHandlesMalformedAndUnderscoredKeys(t *testing.T) {
	i18n := NewI18nParser(nil)
	i18n.dataRows = [][]string{
		{"malformed", "one"},
		{"Sheet_Field_With_Underscore_10", "ten"},
		{"Sheet_Field_With_Underscore_2", "two"},
	}
	i18n.sortDataRows()
	if i18n.dataRows[0][0] != "Sheet_Field_With_Underscore_2" || i18n.dataRows[1][0] != "Sheet_Field_With_Underscore_10" || i18n.dataRows[2][0] != "malformed" {
		t.Fatalf("unexpected i18n order: %#v", i18n.dataRows)
	}
}

func TestWriteI18nPreservesExistingWorkbookStructure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "I18N.xlsx")
	book := excelize.NewFile()
	if err := book.SetSheetName("Sheet1", I18nSheetName); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewSheet("Notes"); err != nil {
		t.Fatal(err)
	}
	style, err := book.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Color: []string{"#FF0000"}, Pattern: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if err := book.SetCellStyle(I18nSheetName, "A4", "A4", style); err != nil {
		t.Fatal(err)
	}
	if err := book.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	if err := book.Close(); err != nil {
		t.Fatal(err)
	}

	i18n := NewI18nParser(nil)
	i18n.SetData("Item_Name_1", "item")
	if err := i18n.WriteToExcel(path); err != nil {
		t.Fatal(err)
	}

	written, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = written.Close() }()
	if index, err := written.GetSheetIndex("Notes"); err != nil || index == -1 {
		t.Fatalf("existing sheet was not preserved: index=%d err=%v", index, err)
	}
	writtenStyle, err := written.GetCellStyle(I18nSheetName, "A4")
	if err != nil || writtenStyle != style {
		t.Fatalf("existing style was not preserved: got=%d want=%d err=%v", writtenStyle, style, err)
	}
}
