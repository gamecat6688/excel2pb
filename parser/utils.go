package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimeCost 耗时统计函数
// 使用方法：defer TimeCost("funcName")()
func TimeCost(tag string) func() {
	start := time.Now()
	return func() {
		tc := time.Since(start)
		fmt.Printf("%v - [%v] time cost = %v\n", time.Now(), tag, tc)
	}
}

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

func ToFloat32(val string) float32 {
	rv, err := strconv.ParseFloat(val, 10)
	if err != nil {
		return 0
	}

	return float32(rv)
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

// MakeI18nKey key = 表名_字段名_主键值
func MakeI18nKey(sheetName string, headName string, keyValue interface{}) string {
	return fmt.Sprintf("%v_%v_%v", sheetName, headName, ToString(keyValue))
}
