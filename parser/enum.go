package parser

import (
	"excel2pb/config"
	"github.com/xuri/excelize/v2"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
)

/*
枚举表处理
*/

type EnumInfo struct {
	Name    string // 枚举名
	Value   string // 枚举值
	Comment string // 注释
}

type EnumParser struct {
	sheetName string // 工作表名
	name      string // 枚举包名
	enums     []EnumInfo
}

func NewEnumParser(sheetName string) *EnumParser {
	enumName := SplitEnumName(sheetName)
	return &EnumParser{
		sheetName: sheetName,
		name:      enumName,
	}
}

func (s *EnumParser) Parse(f *excelize.File) bool {
	rows, _ := f.GetRows(s.sheetName)
	for i, row := range rows {
		if i == 0 {
			// 跳过表头
			continue
		}

		if len(row) == 0 {
			// 跳过空行
			continue
		}

		enum := EnumInfo{
			Name:  row[0],
			Value: row[1],
		}

		// 有注释才处理
		if len(row) > 2 {
			enum.Comment = row[2]
		}

		s.enums = append(s.enums, enum)
	}

	return true
}

func (s *EnumParser) getProtoOutPath(filter string) string {
	return config.Cfg.Outs[FilterFullName[filter]].ProtoPath
}

func (s *EnumParser) getProtoFilePath(filter string) string {
	return filepath.Join(s.getProtoOutPath(filter), s.getProtoName())
}

func (s *EnumParser) getProtoName() string {
	return s.name + ".proto"
}

func (s *EnumParser) ExportProto(filter string) {
	// 解析模板
	tmpl, err := template.New("proto").Parse(ProtoEnumTemplate)
	if err != nil {
		slog.Error("parseFromFile proto enum template fail", "error", err)
		return
	}

	m := &ProtoEnumModel{
		PackageName: config.Cfg.Outs[FilterFullName[filter]].PackageName,
		MessageName: s.name,
	}

	// 数据驱动模板
	for _, v := range s.enums {
		m.Fields = append(m.Fields, FieldModel{
			FieldName: v.Name,
			FieldTag:  ToInt32(v.Value),
			Comment:   v.Comment,
		})
	}

	// 创建proto文件
	outPath := s.getProtoOutPath(filter)
	os.MkdirAll(outPath, os.ModePerm)

	f, err := os.Create(s.getProtoFilePath(filter))
	if err != nil {
		slog.Error("create proto file fail", "error", err)
		return
	}

	// 执行模板,输出文件
	err = tmpl.Execute(f, m)
	if err != nil {
		slog.Error("tmpl.Execute fail", "error", err)
		return
	}
}
