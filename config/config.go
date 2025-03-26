package config

const (

	// excel原始表格路径
	ExcelDir = "assets/xls/"

	// 导出的proto目录
	ClientOutProtoPath = "assets/out_proto/client/"
	ServerOutProtoPath = "assets/out_proto/server/"

	// proto的import路径
	ProtoImportPath = "" // /结尾
	// 客户端proto包名
	ClientProtoPackage = "pb"
	// 服务器proto包名
	ServerProtoPackage = "pbs"

	// 导出的数据目录
	OutDataPath = "assets/out_data/"

	// 输出代码的路径
	OutCodeCppPath    = "assets/out_code/cpp/"
	OutCodeCsharpPath = "assets/out_code/csharp/"
	OutCodeJavaPath   = "assets/out_code/java/"
	OutCodeGolangPath = "assets/out_code/golang/"

	// 代码模板的输入路径
	TplCodeCppPath    = "assets/template/cpp/"
	TplCodeCsharpPath = "assets/template/csharp/"
	TplCodeJavaPath   = "assets/template/java/"
	TplCodeGolangPath = "assets/template/golang/"
)
