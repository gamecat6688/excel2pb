package parser

import "strings"

// SplitEnumName 分割枚举表名, 去掉Enum后缀
// 如: ItemType_Enum -> ItemType
func SplitEnumName(sheetName string) string {
	ss := strings.Split(sheetName, "_")
	return ss[0]
}

// DataRowIndex2ExcelRow 数据的行下标，转换为excel的行数
func DataRowIndex2ExcelRow(rowIndex int32) int32 {
	return DataRow2ExcelRow(rowIndex) + 1
}

// DataRow2ExcelRow 数据的行下标，转换为excel的行数
func DataRow2ExcelRow(row int32) int32 {
	return row + HeadCount
}
