package parser

import (
	"path/filepath"
	"testing"
)

func TestCodeGeneratorsRejectMissingTemplates(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	sheet := makeSheet("Item", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"},
	})
	root.sheets[sheet.sheetName] = sheet
	templatePath := filepath.Join(t.TempDir(), "missing")
	outputPath := t.TempDir()

	tests := []struct {
		name   string
		loader func() bool
		module func() bool
	}{
		{
			name:   "golang",
			loader: func() bool { return NewGolangLoaderCode(templatePath, outputPath).GenCode(root) },
			module: func() bool { return NewGolangModuleCode(templatePath, outputPath).GenCode(root, sheet) },
		},
		{
			name:   "csharp",
			loader: func() bool { return NewCsharpLoaderCode(templatePath, outputPath).GenCode(root) },
			module: func() bool { return NewCsharpModuleCode(templatePath, outputPath).GenCode(root, sheet) },
		},
		{
			name:   "godot",
			loader: func() bool { return NewGodotLoaderCode(templatePath, outputPath).GenCode(root) },
			module: func() bool { return NewGodotModuleCode(templatePath, outputPath).GenCode(root, sheet) },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.loader() {
				t.Fatal("loader generation must fail when no loader template exists")
			}
			if test.module() {
				t.Fatal("module generation must fail when no module template exists")
			}
		})
	}
}

func TestTimestampPrimaryKeyTypeMappings(t *testing.T) {
	if GolangTypeMap[TimestampName] != "int64" {
		t.Fatalf("golang timestamp key type = %q", GolangTypeMap[TimestampName])
	}
	if CSharpTypeMap[TimestampName] != "long" {
		t.Fatalf("csharp timestamp key type = %q", CSharpTypeMap[TimestampName])
	}
}
