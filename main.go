package main

import (
	"excel2pb/parser"
)

func main() {
	p := parser.NewParser()
	p.ParseExcels()
	p.Export()

	//if RewriteI18nExcel {
	//	// 回写多语言表格
	//	lib.I18n.WriteToExcel(i18nExcelFile)
	//}
	//
	//// 输出多语言文本
	//jsonFile := fmt.Sprintf("%v/%v.json", jsonFileDir, lib.I18n.GetName())
	//lib.I18n.WriteToJson(jsonFile)

	println("exit app")
}
