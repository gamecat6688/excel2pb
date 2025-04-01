package data

type GameData struct {
{{range .Names}}	{{.}} *{{.}}Model
{{end}} }

func (m *GameData) loadAllModel() {
{{range .Names}}	m.{{.}} = New{{.}}Model()
{{end}} }
