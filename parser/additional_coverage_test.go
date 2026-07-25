package parser

import (
	"strings"
	"testing"
)

func TestValidationTypeConversionAndDependencies(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	ref := makeSheet("Ref", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"},
	})
	child := makeSheet("Child", [][]string{
		{"name"}, {"string"}, {"cs"}, {""}, {"nested"},
	})
	item := makeSheet("Item", [][]string{
		{"ID", "Flag", "Values", "Children", "Quality", "CreatedAt", "Text", "UniqueName", "RefID", "Index"},
		{"pk int32", "bool", "repeated int32", "repeated Child", "Quality", "timestamp", "i18n", "unique string", "int32", "string"},
		{"cs", "cs", "cs", "cs", "cs", "cs", "cs", "cs", "cs", "cs"},
		{"", "", "", "", "", "", "", "", "", ""},
		{"1", "true", "2;3", "one;two", "Rare", "2025-03-01 08:00:05", "text", "only", "1", "x"},
	})
	item.headers[8].tags = []HeadTag{"fk:Ref.ID"}
	item.headers[9].tags = []HeadTag{"index"}
	quality := &EnumParser{sheetName: "Quality_Enum", name: "Quality", enums: []EnumInfo{{Name: "Rare", Value: "2"}}}
	root.sheets[ref.sheetName] = ref
	root.sheets[child.sheetName] = child
	root.sheets[item.sheetName] = item
	root.enums[quality.name] = quality

	item.checks(root)
	if !item.GetFiled(3).IsCustomMessage(root) || !item.GetFiled(4).IsCustomEnum(root) || item.GetFiled(0).HasTags() {
		t.Fatal("custom type or tag detection failed")
	}
	if got := item.getImportMessages(root); strings.Join(got, ",") != "Child,Quality" {
		t.Fatalf("imports = %#v", got)
	}
	if got := item.getAllProtoFiles(root); strings.Join(got, ",") != "Item.proto,Child.proto,Quality.proto" {
		t.Fatalf("proto files = %#v", got)
	}

	if values := item.procBaseProtoType(root, item.GetFiled(2), 0, "2;3"); len(values) != 2 || values[1].Int() != 3 {
		t.Fatalf("base values = %#v", values)
	}
	if item.TypeNameToValue(root, item.GetFiled(1), 0, "true").Bool() != true || item.TypeNameToValue(root, item.GetFiled(4), 0, "Rare").Enum() != 2 {
		t.Fatal("basic or enum value conversion failed")
	}
	if item.TypeNameToValue(root, item.GetFiled(5), 0, "2025-03-01 08:00:05").Int() != 1740787205 {
		t.Fatal("timestamp conversion failed")
	}
	if item.TypeNameToValue(root, item.GetFiled(6), 0, "text").String() != "Item_Text_1" {
		t.Fatal("i18n key conversion failed")
	}

	files, err := parseProtoFile([]string{"record.proto"}, []string{"testdata"})
	if err != nil {
		t.Fatal(err)
	}
	childType, err := getMessageType(files, "test.Child")
	if err != nil {
		t.Fatal(err)
	}
	custom := item.procCustomProtoType(root, childType, item.GetFiled(3), 0, "one;two")
	if len(custom) != 2 || custom[0].Message().Get(childType.Descriptor().Fields().ByName("name")).String() != "one" {
		t.Fatalf("custom values = %#v", custom)
	}

	unsupported := Head{info: [HeadCount]string{HeadName: "Bad", HeadType: "unknown"}}
	func() {
		defer func() {
			if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "unsupported type \"unknown\"") {
				t.Fatalf("unexpected unknown-type panic: %v", recovered)
			}
		}()
		item.TypeNameToValue(root, unsupported, 0, "bad")
	}()
}

func TestUniqueValidationReportsDuplicate(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"Name"}, {"unique string"}, {"cs"}, {""}, {"same"}, {"same"},
	})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "sheet=\"Item\"") || !strings.Contains(recovered.(string), "field=\"Name\"") {
			t.Fatalf("unexpected uniqueness error: %v", recovered)
		}
	}()
	s.checkUnique(NewParser())
}

func TestMissingForeignKeyTargetDoesNotCrashValidation(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"RefID"}, {"int32"}, {"cs"}, {""}, {"1"},
	})
	s.SetSourceFile("assets/xls/item.xlsx")
	s.headers[0].tags = []HeadTag{"fk:Missing.ID"}
	s.checkTags(NewParser())
}
