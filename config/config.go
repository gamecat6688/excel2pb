package config

var (

	// ExcelDir excel原始表格路径
	ExcelDir = "assets/xls/"

	// ProtoOutPaths 导出的proto目录
	ProtoOutPaths = map[string]string{
		"c": "assets/out_proto/client/",
		"s": "assets/out_proto/server/",
	}

	// DataOutPaths 导出的数据目录
	DataOutPaths = map[string]string{
		"c": "assets/out_data/client/",
		"s": "assets/out_data/server/",
	}

	// DataExtensions 导出的数据后缀
	DataExtensions = map[string]string{
		"c": ".data",
		"s": ".data",
	}

	// ProtoImportPath proto的import路径
	ProtoImportPath = "" // "/"结尾

	// ProtoPackages proto包名
	ProtoPackages = map[string]string{
		"c": "pb",
		"s": "pbs",
	}

	// GenerateLanguage 生成代码的语言
	GenerateLanguage = map[string]string{
		"c": "csharp",
		"s": "golang",
	}

	// CodeOutPaths 输出代码的路径
	CodeOutPaths = map[string]string{
		"cpp":    "assets/out_code/cpp/",
		"csharp": "assets/out_code/csharp/",
		"java":   "assets/out_code/java/",
		"golang": "assets/out_code/golang/",
	}

	// TplCodePaths 代码模板的输入路径
	TplCodePaths = map[string]string{
		"cpp":    "assets/template/cpp/",
		"csharp": "assets/template/csharp/",
		"java":   "assets/template/java/",
		"golang": "assets/template/golang/",
	}
)
