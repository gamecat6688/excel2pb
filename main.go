package main

import (
	"excel2pb/config"
	"excel2pb/parser"
	"fmt"
	"log/slog"
	"runtime"
)

func main() {
	config.LoadConfig()

	if config.Cfg.MaxProcess > 0 {
		runtime.GOMAXPROCS(config.Cfg.MaxProcess)
	}

	slog.Info(fmt.Sprintf("max process %v", runtime.GOMAXPROCS(0)))

	p := parser.NewParser()
	p.ParseExcels()
	p.MergeI18n()
	p.Export()

	println("exit app")
}
