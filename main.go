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

	println("exit app")
}
