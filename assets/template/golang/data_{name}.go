package data

import {{.PackageName}} "{{.GoPackagePath}}"

type {{.Name}} struct {
	*{{.FullName}}
}
{{if .MultiKey}}
// {{.KeyType}} {{.Name}} 的组合主键
type {{.KeyType}} struct {
{{range .Keys}}	{{.Name}} {{.Type}}
{{end}}}
{{end}}
func Get{{.Name}}Model() *{{.Name}}Model {
	return getGameData().{{.Name}}
}

func New{{.Name}}Model() *{{.Name}}Model {
	m := &{{.Name}}Model{
		DefaultModel: NewDefaultModel[{{.KeyType}}, *{{.Name}}](),
	}

	cfg := &{{.FullName}}Config{}
	load(cfg)
	for _, v := range cfg.Records {
		m.rows[{{if .MultiKey}}{{.KeyType}}{ {{range $i, $k := .Keys}}{{if $i}}, {{end}}v.{{$k.Name}}{{end}} }{{else}}v.{{.KeyName}}{{end}}] = &{{.Name}}{v}
	}

	m.onInit()
	return m
}

type {{.Name}}Model struct {
	*DefaultModel[{{.KeyType}}, *{{.Name}}]
}

func (m *{{.Name}}Model) onInit() {

}
