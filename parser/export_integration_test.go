package parser

import (
	"excel2pb/config"
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
			"Server": {ProtoPath: filepath.Join(dir, "proto", "server"), DataPath: filepath.Join(dir, "data", "server"), DataExt: ".data", PbPath: filepath.Join(dir, "pb", "server"), PackageName: "test", CodeLanguage: "golang"},
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

	root := NewParser()
	attr := makeSheet("Attr", [][]string{
		{"RaceID", "Type", "Value"},
		{"pk int32", "pk string", "int64"},
		{"c", "c", "c"},
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
