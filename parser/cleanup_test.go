package parser

import (
	"excel2pb/config"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupStaleOutputsRemovesOnlyManagedFiles(t *testing.T) {
	dir := configureTestExports(t)
	for path, content := range map[string]string{
		filepath.Join(dir, "templates", "golang", "loader.go"):      "loader",
		filepath.Join(dir, "templates", "golang", "data_{name}.go"): "module",
		filepath.Join(dir, "templates", "csharp", "loader.cs"):      "loader",
		filepath.Join(dir, "templates", "csharp", "data_{Name}.cs"): "module",
	} {
		writeTestFile(t, path, content)
	}

	root := NewParser()
	item := makeSheet("Item", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"},
	})
	root.sheets[item.sheetName] = item
	root.enums["Quality"] = &EnumParser{name: "Quality"}
	root.enums["item_quality"] = &EnumParser{name: "item_quality"}

	client := configForFilter(ClientFlag)
	server := configForFilter(ServerFlag)
	files := map[string]bool{
		filepath.Join(client.ProtoPath, generatedFilesManifest):        false,
		filepath.Join(client.ProtoPath, "Item.proto"):                  true,
		filepath.Join(client.ProtoPath, "Old.proto"):                   false,
		filepath.Join(client.ProtoPath, "notes.txt"):                   true,
		filepath.Join(client.DataPath, "Item"+client.DataExt):          true,
		filepath.Join(client.DataPath, "Old"+client.DataExt):           false,
		filepath.Join(client.PbPath, "Item.cs"):                        true,
		filepath.Join(client.PbPath, "Quality.cs"):                     true,
		filepath.Join(client.PbPath, "ItemQuality.cs"):                 true,
		filepath.Join(client.PbPath, "item_quality.cs"):                false,
		filepath.Join(client.PbPath, "Old.cs"):                         false,
		filepath.Join(server.PbPath, server.PackageName, "Item.pb.go"): true,
		filepath.Join(server.PbPath, server.PackageName, "Old.pb.go"):  false,
		filepath.Join(configForCode(ClientFlag), "loader.cs"):          true,
		filepath.Join(configForCode(ClientFlag), "data_Item.cs"):       true,
		filepath.Join(configForCode(ClientFlag), "data_Old.cs"):        false,
		filepath.Join(configForCode(ServerFlag), "loader.go"):          true,
		filepath.Join(configForCode(ServerFlag), "data_item.go"):       true,
		filepath.Join(configForCode(ServerFlag), "data_old.go"):        false,
		filepath.Join(configForCode(ServerFlag), "handwritten.txt"):    true,
	}
	for path := range files {
		writeTestFile(t, path, "test")
	}

	root.cleanupStaleOutputs()
	for path, shouldExist := range files {
		_, err := os.Stat(path)
		if shouldExist && err != nil {
			t.Errorf("expected %s to remain: %v", path, err)
		}
		if !shouldExist && !os.IsNotExist(err) {
			t.Errorf("expected stale file %s to be removed, stat error: %v", path, err)
		}
	}
}

func TestCsharpProtoFilenameMatchesProtocConvention(t *testing.T) {
	tests := map[string]string{
		"item_quality": "ItemQuality.cs",
		"Item_Quality": "ItemQuality.cs",
		"I18N":         "I18N.cs",
		"shop":         "Shop.cs",
	}
	for input, want := range tests {
		if got := csharpProtoFilename(input); got != want {
			t.Errorf("csharpProtoFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestManagedOutputRejectsBroadProtectedDirectories(t *testing.T) {
	for _, root := range []string{os.TempDir()} {
		t.Run(root, func(t *testing.T) {
			if _, err := getManagedOutput(map[string]*managedOutput{}, root); err == nil {
				t.Fatalf("getManagedOutput(%q) accepted a broad protected directory", root)
			}
		})
	}
}

func configForFilter(filter string) config.OutConfig {
	return config.Cfg.Outs[FilterFullName[filter]]
}

func configForCode(filter string) string {
	cfg := configForFilter(filter)
	return config.Cfg.CodeOutPaths[cfg.CodeLanguage]
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
