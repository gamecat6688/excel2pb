package parser

import "testing"

func TestValueConversionsAndSplitting(t *testing.T) {
	if !IsNumber("-42") || IsNumber("4.2") {
		t.Fatal("IsNumber did not distinguish integer input")
	}
	if ToInt("bad") != 0 || ToInt32("12") != 12 || ToInt64("9000000000") != 9000000000 {
		t.Fatal("integer conversion returned unexpected values")
	}
	if ToFloat32("1.5") != 1.5 || ToFloat64("2.25") != 2.25 || ToBool("true") != true || ToBool("bad") {
		t.Fatal("scalar conversion returned unexpected values")
	}

	base := SplitBaseValue(" one; ;two ;")
	if len(base) != 2 || base[0] != "one" || base[1] != "two" {
		t.Fatalf("unexpected base values: %#v", base)
	}
	custom := SplitCustomValue("1|2; 3|4 ;")
	if len(custom) != 2 || len(custom[1]) != 2 || custom[1][1] != "4" {
		t.Fatalf("unexpected custom values: %#v", custom)
	}
}

func TestFormattingHelpers(t *testing.T) {
	if got := SplitEnumName("ItemType_Enum"); got != "ItemType" {
		t.Fatalf("SplitEnumName = %q", got)
	}
	if got := SplitEnumName("Item_Quality_Enum"); got != "Item_Quality" {
		t.Fatalf("SplitEnumName with underscores = %q", got)
	}
	if got := DataRow2ExcelRow(0); got != 4 {
		t.Fatalf("DataRow2ExcelRow = %d", got)
	}
	if got := DataRowIndex2ExcelRow(0); got != 5 {
		t.Fatalf("DataRowIndex2ExcelRow = %d", got)
	}
	if got := DataTimeToRFC3339("2025-03-01 08:00:05", "+08:00"); got != "2025-03-01T08:00:05+08:00" {
		t.Fatalf("DataTimeToRFC3339 = %q", got)
	}
	if got := MakeI18nKey("Item", "Name", 1001); got != "Item_Name_1001" {
		t.Fatalf("MakeI18nKey = %q", got)
	}
}
