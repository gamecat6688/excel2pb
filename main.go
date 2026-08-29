package main

import (
	"embed"
	"fmt"
	"log/slog"
	"runtime"

	"excel2pb/config"
	"excel2pb/parser"
	"excel2pb/works"
)

//go:embed assets/template/*/*
var builtInTemplates embed.FS

func main() {
	config.LoadConfig()
	parser.SetEmbeddedTemplates(builtInTemplates)
	if err := parser.ConfigureFilters(config.Cfg.Outs); err != nil {
		panic(fmt.Sprintf("invalid export configuration: %v", err))
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(config.Cfg.LogLevel)); err != nil {
		panic(fmt.Sprintf("invalid log level %q: %v", config.Cfg.LogLevel, err))
	}
	slog.SetLogLoggerLevel(logLevel)

	if config.Cfg.MaxProcess > 0 {
		runtime.GOMAXPROCS(config.Cfg.MaxProcess)
	}
	works.SetLimit(runtime.GOMAXPROCS(0))

	slog.Info(fmt.Sprintf("max process %v", runtime.GOMAXPROCS(0)))

	p := parser.NewParser()
	p.ParseExcels()
	p.MergeI18n()
	p.Export()
}
