package data

import "server/pbs"

type {{.Name}} struct {
	*pbs.{{.Name}}
}

func Get{{.Name}}Model() *{{.Name}}Model {
	return getGameData().{{.Name}}
}

func New{{.Name}}Model() *{{.Name}}Model {
	m := &{{.Name}}Model{
		DefaultModel: NewDefaultModel[{{.KeyType}}, *{{.Name}}](),
	}

	cfg := &pbs.{{.Name}}Config{}
	load(cfg)
	for _, v := range cfg.Records {
		m.rows[v.{{.KeyName}}] = &{{.Name}}{v}
	}

	m.onInit()
	return m
}

type {{.Name}}Model struct {
	*DefaultModel[{{.KeyType}}, *{{.Name}}]
}

func (m *{{.Name}}Model) onInit() {

}
