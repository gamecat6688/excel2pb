package parser

import (
	"fmt"
	"google.golang.org/protobuf/types/descriptorpb"
	"strings"
)

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

func ProtoType(goType string) (*descriptorpb.FieldDescriptorProto_Type, error) {
	switch goType {
	case "string":
		return descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), nil
	case "int32":
		return descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), nil
	case "int64":
		return descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(), nil
	case "float":
		return descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), nil
	case "double":
		return descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), nil
	case "bool":
		return descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(), nil
	default:
		return nil, fmt.Errorf("不支持的类型: %s", goType)
	}
}
