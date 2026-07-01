package parser

import (
	"strings"
	"testing"
	"text/template"
)

// 用合成的多主键模型渲染真实模板，验证输出可读且语法合理
func renderTpl(t *testing.T, path string, m *CodeModuleModel) string {
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		t.Fatalf("parse %v: %v", path, err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, m); err != nil {
		t.Fatalf("exec %v: %v", path, err)
	}
	return sb.String()
}

func TestMultiKeyGolangTemplate(t *testing.T) {
	m := &CodeModuleModel{
		Name:     "Attr",
		FullName: "pbs.Attr",
		KeyType:  "AttrKey",
		KeyName:  "RaceID",
		MultiKey: true,
		Keys: []KeyField{
			{Type: "int32", Name: "RaceID"},
			{Type: "pbs.AttrType", Name: "Type"},
		},
	}
	out := renderTpl(t, "../assets/template/golang/data_{name}.go", m)
	t.Logf("\n%s", out)

	for _, want := range []string{
		"type AttrKey struct {",
		"RaceID int32",
		"Type pbs.AttrType",
		"NewDefaultModel[AttrKey, *Attr]()",
		"m.rows[AttrKey{ v.RaceID, v.Type }] = &Attr{v}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("golang multi-key output missing: %q", want)
		}
	}
}

func TestMultiKeyCsharpTemplate(t *testing.T) {
	m := &CodeModuleModel{
		Name:     "Attr",
		KeyType:  "(int, long)",
		KeyName:  "RaceID",
		MultiKey: true,
		Keys: []KeyField{
			{Type: "int", Name: "RaceID"},
			{Type: "long", Name: "Type"},
		},
	}
	out := renderTpl(t, "../assets/template/csharp/{Name}.cs", m)
	t.Logf("\n%s", out)

	for _, want := range []string{
		"Dictionary<(int, long), Attr>",
		"rows[(r.RaceID, r.Type)] = r;",
		"public Attr Get((int, long) key)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("csharp multi-key output missing: %q", want)
		}
	}
}

// 单主键必须保持与旧版一致（向后兼容）
func TestSingleKeyGolangTemplateUnchanged(t *testing.T) {
	m := &CodeModuleModel{
		Name:     "Item",
		FullName: "pbs.Item",
		KeyType:  "int32",
		KeyName:  "ID",
		MultiKey: false,
		Keys:     []KeyField{{Type: "int32", Name: "ID"}},
	}
	out := renderTpl(t, "../assets/template/golang/data_{name}.go", m)
	if !strings.Contains(out, "m.rows[v.ID] = &Item{v}") {
		t.Errorf("single-key golang output changed:\n%s", out)
	}
	if strings.Contains(out, "struct {\n\tID") || strings.Contains(out, "ItemKey struct") {
		t.Errorf("single-key should NOT emit composite key struct:\n%s", out)
	}
}
