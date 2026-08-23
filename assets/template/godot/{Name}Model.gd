class_name {{.Name}}Model
extends RefCounted

const Proto = preload("{{.ProtoScriptPath}}")
const DEFAULT_DATA_FILE := "{{.DataFilePath}}"

var rows: Dictionary = {}


func load_data(path: String = "") -> bool:
	var data_path := path
	if data_path.is_empty():
		data_path = get_script().resource_path.get_base_dir().path_join(DEFAULT_DATA_FILE).simplify_path()

	if not FileAccess.file_exists(data_path):
		push_error("{{.Name}} config file does not exist: %s" % data_path)
		return false

	var data := FileAccess.get_file_as_bytes(data_path)
	var config := Proto.{{.Name}}Config.new()
	var consumed: int = config.ParseFromBytes(data)
	if consumed != data.size():
		push_error("{{.Name}} config parse failed: consumed %d of %d bytes" % [consumed, data.size()])
		return false

	rows.clear()
	for row in config.Records():
		rows[{{if .MultiKey}}[{{range $i, $k := .Keys}}{{if $i}}, {{end}}row.{{$k.Name}}{{end}}]{{else}}row.{{.KeyName}}{{end}}] = row
	return true

{{if .MultiKey}}
static func make_key({{range $i, $k := .Keys}}{{if $i}}, {{end}}{{$k.Name}}: {{$k.Type}}{{end}}) -> Array:
	return [{{range $i, $k := .Keys}}{{if $i}}, {{end}}{{$k.Name}}{{end}}]

{{end}}
func has(key: {{.KeyType}}) -> bool:
	return rows.has(key)


func get_row(key: {{.KeyType}}):
	return rows.get(key)


func get_rows() -> Dictionary:
	return rows
