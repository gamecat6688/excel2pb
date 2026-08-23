class_name GameData
extends RefCounted

var models: Dictionary = {}


func load_all() -> bool:
	models.clear()
{{range .Names}}	if not _load_model("{{.}}", {{.}}Model.new()):
		return false
{{end}}	return true


func _load_model(model_name: StringName, model) -> bool:
	if not model.load_data():
		return false
	models[model_name] = model
	return true


func get_model(model_name: StringName):
	return models.get(model_name)
