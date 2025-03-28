package parser

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"strconv"
	"strings"
)

func ToString(val interface{}) string {
	return fmt.Sprintf("%v", val)
}

func ToInt(val string) int {
	rv, err := strconv.ParseInt(val, 10, 0)
	if err != nil {
		return 0
	}

	return int(rv)
}

func ToInt32(val string) int32 {
	rv, err := strconv.ParseInt(val, 10, 0)
	if err != nil {
		return 0
	}

	return int32(rv)
}

func ToInt64(val string) int64 {
	rv, err := strconv.ParseInt(val, 10, 0)
	if err != nil {
		return 0
	}

	return rv
}

// SplitEnumName 分割枚举表名, 去掉Enum后缀
// 如: ItemType_Enum -> ItemType
func SplitEnumName(sheetName string) string {
	ss := strings.Split(sheetName, "_")
	return ss[0]
}

// SplitBaseValue 一维数组
// ab;df;asd
func SplitBaseValue(str string) (rv []string) {
	ss := strings.Split(str, ";")
	for _, v := range ss {
		val := strings.Trim(v, " ")
		if len(val) == 0 {
			continue
		}

		rv = append(rv, val)
	}
	return
}

// SplitCustomValue 二维数组
// 100|1;101|2;103|5
func SplitCustomValue(str string) (rv [][]string) {
	abab := strings.Split(str, ";")
	for _, ab := range abab {
		val := strings.Trim(ab, " ")
		if len(val) == 0 {
			continue
		}

		ss := strings.Split(ab, "|")
		rv = append(rv, ss)
	}
	return
}

// DataRowIndex2ExcelRow 数据的行下标，转换为excel的行数
func DataRowIndex2ExcelRow(rowIndex int32) int32 {
	return DataRow2ExcelRow(rowIndex) + 1
}

// DataRow2ExcelRow 数据的行下标，转换为excel的行数
func DataRow2ExcelRow(row int32) int32 {
	return row + HeadCount
}

func TypeNameToValue(root *Parser, fdDesc *desc.FieldDescriptor, sheetName string, headName string, typeName string, value string) interface{} {
	switch typeName {
	case "string", I18nName:
		v := value
		return v
	case "int32":
		v, _ := strconv.ParseInt(value, 10, 23)
		return int32(v)
	case "int64":
		v, _ := strconv.ParseInt(value, 10, 64)
		return v
	case "float", "double":
		v, _ := strconv.ParseFloat(value, 10)
		return v
	case "bool":
		v, _ := strconv.ParseBool(value)
		return v
	default:
		if root.hasEnumParser(typeName) {
			v, _ := strconv.ParseInt(value, 10, 64)
			enumDesc := fdDesc.GetEnumType().FindValueByNumber(int32(v))
			return enumDesc
		} else {
			panic(fmt.Sprintf("[%v.%v]not support type %v", sheetName, headName, typeName))
			return nil
		}
	}
	return 0
}

//func ProtoType(goType string) (*descriptorpb.FieldDescriptorProto_Type, error) {
//	switch goType {
//	case "string":
//		return descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), nil
//	case "int32":
//		return descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), nil
//	case "int64":
//		return descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(), nil
//	case "float":
//		return descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), nil
//	case "double":
//		return descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), nil
//	case "bool":
//		return descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(), nil
//	default:
//		return nil, fmt.Errorf("不支持的类型: %s", goType)
//	}
//}
