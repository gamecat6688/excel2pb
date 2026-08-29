package parser

import (
	"excel2pb/config"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	sourceFile string
	sheetName  string // 工作表名
	name       string // 枚举包名
	enums      []EnumInfo
}

func NewEnumParser(sheetName string) *EnumParser {
	enumName := SplitEnumName(sheetName)
	return &EnumParser{
		sheetName: sheetName,
		name:      enumName,
	}
}

func (s *EnumParser) SetSourceFile(sourceFile string) {
	s.sourceFile = sourceFile
}

func (s *EnumParser) Parse(f *excelize.File) bool {
	if !strings.HasSuffix(s.sheetName, "_Enum") || !isValidProtobufIdentifier(s.name) || !isStableGeneratedIdentifier(s.name) {
		slog.Error("invalid enum sheet name", "file", s.sourceFile, "sheet", s.sheetName, "enum", s.name)
		return false
	}
	rows, err := f.GetRows(s.sheetName)
	if err != nil {
		slog.Error("read enum sheet rows failed", "file", s.sourceFile, "sheet", s.sheetName, "error", err)
		return false
	}
	s.enums = nil
	names := map[string]struct{}{}
	values := map[int32]struct{}{}
	for i, row := range rows {
		if i == 0 {
			// 跳过表头
			continue
		}

		if len(row) == 0 {
			// 跳过空行
			continue
		}
		if len(row) < 2 {
			slog.Error("invalid enum row", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1, "column_count", len(row), "expected_columns", 2)
			return false
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			slog.Error("enum name is empty", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1)
			return false
		}
		if !isValidProtobufIdentifier(name) {
			slog.Error("invalid enum value name", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1, "name", name)
			return false
		}
		value, err := parseEnumValue(row[1])
		if err != nil {
			slog.Error("invalid enum value", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1, "value", row[1], "error", err)
			return false
		}
		if len(s.enums) == 0 && value != 0 {
			slog.Error("first enum value must be zero", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1, "value", value)
			return false
		}
		if _, exists := names[name]; exists {
			slog.Error("duplicate enum name", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1, "name", name)
			return false
		}
		if _, exists := values[value]; exists {
			slog.Error("duplicate enum value", "file", s.sourceFile, "sheet", s.sheetName, "excel_row", i+1, "value", value)
			return false
		}

		enum := EnumInfo{
			Name:  name,
			Value: strconv.FormatInt(int64(value), 10),
		}

		// 有注释才处理
		if len(row) > 2 {
			enum.Comment = row[2]
		}

		s.enums = append(s.enums, enum)
		names[name] = struct{}{}
		values[value] = struct{}{}
	}
	if len(s.enums) == 0 {
		slog.Error("enum sheet has no values", "file", s.sourceFile, "sheet", s.sheetName)
		return false
	}

	return true
}

// 获取枚举值
func (s *EnumParser) getEnumValue(enumName string) int32 {
	for _, v := range s.enums {
		if v.Name == enumName {
			value, err := parseEnumValue(v.Value)
			if err != nil {
				panic(fmt.Sprintf("[file=%q sheet=%q enum=%q] invalid enum value %q: %v", s.sourceFile, s.sheetName, s.name, v.Value, err))
			}
			return value
		}
	}

	panic(fmt.Sprintf("[file=%q sheet=%q enum=%q] enum value %q not found", s.sourceFile, s.sheetName, s.name, enumName))
}

func (s *EnumParser) hasEnumValue(enumValue int32) bool {
	for _, v := range s.enums {
		value, err := parseEnumValue(v.Value)
		if err != nil {
			panic(fmt.Sprintf("[file=%q sheet=%q enum=%q] invalid enum value %q: %v", s.sourceFile, s.sheetName, s.name, v.Value, err))
		}
		if value == enumValue {
			return true
		}
	}
	return false
}

func parseEnumValue(value string) (int32, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(parsed), nil
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
		panic(fmt.Sprintf("[file=%q sheet=%q] parse enum proto template failed: %v", s.sourceFile, s.sheetName, err))
	}

	m := &ProtoEnumModel{
		PackageName:   config.Cfg.Outs[FilterFullName[filter]].PackageName,
		GoPackagePath: goPackagePathForFilter(filter),
		MessageName:   s.name,
	}

	// 数据驱动模板
	for _, v := range s.enums {
		value, err := parseEnumValue(v.Value)
		if err != nil {
			panic(fmt.Sprintf("[file=%q sheet=%q enum=%q] invalid enum value %q: %v", s.sourceFile, s.sheetName, s.name, v.Value, err))
		}
		m.Fields = append(m.Fields, FieldModel{
			FieldName: v.Name,
			FieldTag:  value,
			Comment:   v.Comment,
		})
	}

	// 创建proto文件
	outPath := s.getProtoOutPath(filter)
	if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
		panic(fmt.Sprintf("[file=%q sheet=%q] create enum proto output directory %q failed: %v", s.sourceFile, s.sheetName, outPath, err))
	}

	f, err := os.Create(s.getProtoFilePath(filter))
	if err != nil {
		panic(fmt.Sprintf("[file=%q sheet=%q] create enum proto file %q failed: %v", s.sourceFile, s.sheetName, s.getProtoFilePath(filter), err))
	}

	// 执行模板,输出文件
	err = tmpl.Execute(f, m)
	if err != nil {
		_ = f.Close()
		panic(fmt.Sprintf("[file=%q sheet=%q] render enum proto file %q failed: %v", s.sourceFile, s.sheetName, s.getProtoFilePath(filter), err))
	}
	if err := f.Close(); err != nil {
		panic(fmt.Sprintf("[file=%q sheet=%q] close enum proto file %q failed: %v", s.sourceFile, s.sheetName, s.getProtoFilePath(filter), err))
	}
}
