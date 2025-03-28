package main

import (
	"excel2pb/parser"
)

func main() {
	p := parser.NewParser()
	p.ParseExcels()
	p.Export()

	// 测试生成后的反序列化
	//data, err := os.ReadFile("assets/out_data/server/Upgrade.data")
	//if err != nil {
	//	panic(err)
	//}
	//cfg := &pbs.UpgradeConfig{}
	//proto.Unmarshal(data, cfg)
	//fmt.Printf("cfg: %v\n", cfg)
	//
	//itemData, err := os.ReadFile("assets/out_data/server/Item.data")
	//if err != nil {
	//	panic(err)
	//}
	//cfgItem := &pbs.ItemConfig{}
	//proto.Unmarshal(itemData, cfgItem)
	//fmt.Printf("cfgItem: %v\n", cfgItem)

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
