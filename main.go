package main

import (
	"excel2pb/assets/out_pb/server/pbs"
	"excel2pb/config"
	"excel2pb/parser"
	"fmt"
	"google.golang.org/protobuf/proto"
	"os"
)

func main() {
	config.LoadConfig()
	p := parser.NewParser()
	p.ParseExcels()
	p.Export()

	//testReadData()

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

func testReadData() {
	dataItem, err := os.ReadFile("assets/out_data/server/Item.data")
	if err != nil {
		panic(err)
	}
	cfgItem := &pbs.ItemConfig{}
	proto.Unmarshal(dataItem, cfgItem)
	fmt.Printf("cfgItem: %v\n", cfgItem)

	dataUpgrade, err := os.ReadFile("assets/out_data/server/Upgrade.data")
	if err != nil {
		panic(err)
	}
	cfgUpgrade := &pbs.UpgradeConfig{}
	proto.Unmarshal(dataUpgrade, cfgUpgrade)
	fmt.Printf("Upgrade: %v\n", cfgUpgrade)

	dataShop, err := os.ReadFile("assets/out_data/server/Shop.data")
	if err != nil {
		panic(err)
	}
	cfgShop := &pbs.ShopConfig{}
	proto.Unmarshal(dataShop, cfgShop)
	fmt.Printf("cfgShop: %v\n", cfgShop)
}
