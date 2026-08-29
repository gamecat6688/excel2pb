package parser

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestEmbeddedCodeTemplatesCanBeListedAndParsed(t *testing.T) {
	previous := embeddedTemplates
	SetEmbeddedTemplates(fstest.MapFS{
		"assets/template/godot/game_data.gd":   {Data: []byte("loader {{len .Names}}")},
		"assets/template/godot/{Name}Model.gd": {Data: []byte("module {{.Name}}")},
	})
	t.Cleanup(func() { embeddedTemplates = previous })

	loaders, err := findCodeTemplates("embedded://godot", false)
	if err != nil {
		t.Fatalf("find embedded loader templates: %v", err)
	}
	if got := templateBaseName(loaders[0]); got != "game_data.gd" {
		t.Fatalf("loader template = %q", got)
	}

	modules, err := findCodeTemplates("embedded://godot", true)
	if err != nil {
		t.Fatalf("find embedded module templates: %v", err)
	}
	if got := templateBaseName(modules[0]); got != "{Name}Model.gd" {
		t.Fatalf("module template = %q", got)
	}
	tmpl, err := parseCodeTemplate(modules[0])
	if err != nil {
		t.Fatalf("parse embedded module template: %v", err)
	}
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, &CodeModuleModel{Name: "Item"}); err != nil {
		t.Fatalf("execute embedded module template: %v", err)
	}
	if rendered.String() != "module Item" {
		t.Fatalf("rendered template = %q", rendered.String())
	}
}

func TestEmbeddedCodeTemplatesRejectInvalidLanguagePath(t *testing.T) {
	previous := embeddedTemplates
	SetEmbeddedTemplates(fstest.MapFS{})
	t.Cleanup(func() { embeddedTemplates = previous })

	_, err := listCodeTemplates("embedded://../godot")
	if err == nil {
		t.Fatal("invalid embedded template path was accepted")
	}
	if !strings.Contains(err.Error(), "invalid embedded template path") {
		t.Fatalf("unexpected error: %v", err)
	}
}
