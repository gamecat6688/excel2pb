package main

import (
	"excel2pb/config"
	"excel2pb/parser"
)

func main() {
	config.LoadConfig()
	p := parser.NewParser()
	p.ParseExcels()
	p.MergeI18n()
	p.Export()

	testReadData()

	println("exit app")
}

func testReadData() {
	//dataItem, err := os.ReadFile("assets/out_data/server/Item.data")
	//if err != nil {
	//	panic(err)
	//}
	//cfgItem := &pbs.ItemConfig{}
	//proto.Unmarshal(dataItem, cfgItem)
	//fmt.Printf("cfgItem: %v\n", cfgItem)
	//
	//dataUpgrade, err := os.ReadFile("assets/out_data/server/Upgrade.data")
	//if err != nil {
	//	panic(err)
	//}
	//cfgUpgrade := &pbs.UpgradeConfig{}
	//proto.Unmarshal(dataUpgrade, cfgUpgrade)
	//fmt.Printf("Upgrade: %v\n", cfgUpgrade)
	//
	//dataShop, err := os.ReadFile("assets/out_data/server/Shop.data")
	//if err != nil {
	//	panic(err)
	//}
	//cfgShop := &pbs.ShopConfig{}
	//proto.Unmarshal(dataShop, cfgShop)
	//fmt.Printf("cfgShop: %v\n", cfgShop)
}
