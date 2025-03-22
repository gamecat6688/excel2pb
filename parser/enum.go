package parser

import (
	"github.com/xuri/excelize/v2"
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

//type color int32
//
//var Color = struct {
//	Red, Green, Blue color
//}{1, 2, 3}
