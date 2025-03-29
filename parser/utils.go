package parser

import (
	"excel2pb/config"
	"fmt"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strconv"
	"strings"
	"time"
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

func ToFloat64(val string) float64 {
	rv, err := strconv.ParseFloat(val, 10)
	if err != nil {
		return 0
	}

	return rv
}

func ToBool(val string) bool {
	rv, err := strconv.ParseBool(val)
	if err != nil {
		return false
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

// 例子：2025-03-01 08:00:05 转为 2025-03-01T08:00:05+08:00
// zone 时区 +08:00
func DataTimeToRFC3339(datetime string, zone string) string {
	ss := strings.Split(datetime, " ")
	timeOfZone := ss[0] + "T" + ss[1] + zone
	return timeOfZone
}

func TypeNameToValue(root *Parser, sheetName string, headName string, typeName string, value string) protoreflect.Value {
	switch typeName {
	case "string", I18nName:
		return protoreflect.ValueOfString(value)
	case "int32":
		return protoreflect.ValueOfInt32(ToInt32(value))
	case "int64":
		return protoreflect.ValueOfInt64(ToInt64(value))
	case "float", "double":
		return protoreflect.ValueOfFloat64(ToFloat64(value))
	case "bool":
		return protoreflect.ValueOfBool(ToBool(value))
	case TimestampName:
		timeOfZone := DataTimeToRFC3339(value, config.Cfg.TimeZone)
		t, err := time.Parse(time.RFC3339, timeOfZone)
		if err != nil {
			panic(fmt.Sprintf("[%v.%v]parse time fail, val:%v, time:%v", sheetName, headName, value, timeOfZone))
		}
		return protoreflect.ValueOfInt64(t.Unix())
	default:
		if root.hasEnumParser(typeName) {
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(ToInt32(value)))
		} else {
			panic(fmt.Sprintf("[%v.%v]not support type %v", sheetName, headName, typeName))
		}
	}

	return protoreflect.ValueOf(nil)
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
