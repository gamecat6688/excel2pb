package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type OutConfig struct {
	// 导出的proto目录
	ProtoPath string `yaml:"ProtoPath"`

	// 导出的pb目录
	PbPath string `yaml:"PbPath"`

	// 导出的数据目录
	DataPath string `yaml:"DataPath"`

	// 导出的数据文件扩展名
	DataExt string `yaml:"DataExt"`

	// proto包名
	PackageName string `yaml:"PackageName"`

	// 生成代码的语言
	CodeLanguage string `yaml:"CodeLanguage"`
}

type Config struct {
	TimeZone string `yaml:"TimeZone"`

	EnableI18n bool `yaml:"EnableI18n"`

	ExcelDir string `yaml:"ExcelDir"`

	ProtoImportPath string `yaml:"ProtoImportPath"`

	Outs map[string]OutConfig `yaml:"Outs"`

	CodeOutPaths map[string]string `yaml:"CodeOutPaths"`

	TplCodePaths map[string]string `yaml:"TplCodePaths"`
}

var Cfg *Config

func LoadConfig() {
	configPath := "config.yaml"

	cfg := &Config{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg, err = loadConfigFromData([]byte(defaultConfigData))
		if err != nil {
			panic(err)
		}
	} else {
		cfg, err = loadConfigFromData(data)
		if err != nil {
			panic(err)
		}
	}

	Cfg = cfg
}

func loadConfigFromData(data []byte) (*Config, error) {
	cfg := &Config{}
	err := yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

var defaultConfigData = `
TimeZone: "+08:00"
EnableI18n: true
ExcelDir: assets/xls/
ProtoImportPath: ""

Outs:
  Client:
    ProtoPath: assets/out_proto/client/
    PbPath:    assets/out_pb/client/
    DataPath:  assets/out_data/client/
    DataExt:  ".bytes"
    PackageName:  "pb"
    CodeLanguage:  "csharp"
  Server:
    ProtoPath: assets/out_proto/server/
    PbPath: assets/out_pb/server/
    DataPath: assets/out_data/server/
    DataExt: ".data"
    PackageName: "pbs"
    CodeLanguage: "golang"

TplCodePaths:
  golang: assets/template/golang/
  csharp: assets/template/csharp/
  cpp: assets/template/cpp/
  java: assets/template/java/

CodeOutPaths:
  golang: assets/out_code/golang/
  csharp: assets/out_code/csharp/
  cpp:    assets/out_code/cpp/
  java:   assets/out_code/java/

`
