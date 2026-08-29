package parser

import (
	"bytes"
	"log/slog"
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

func TestMissingForeignKeyTargetFailsValidation(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"RefID"}, {"int32"}, {"cs"}, {""}, {"1"},
	})
	s.SetSourceFile("assets/xls/item.xlsx")
	s.headers[0].tags = []HeadTag{"fk:Missing.ID"}
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "target sheet") {
			t.Fatalf("unexpected foreign-key validation result: %v", recovered)
		}
	}()
	s.checkTags(NewParser())
}

func TestEmptyScalarFailsValidation(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"ID", "Enabled"}, {"pk int32", "bool"}, {"cs", "cs"}, {"", ""}, {"1"},
	})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "scalar value is empty") {
			t.Fatalf("unexpected empty-scalar validation result: %v", recovered)
		}
	}()
	s.checks(NewParser())
}

func TestNumericEnumValueMustBeDefined(t *testing.T) {
	root := NewParser()
	root.enums["Quality"] = &EnumParser{name: "Quality", enums: []EnumInfo{{Name: "None", Value: "0"}, {Name: "Rare", Value: "2"}}}
	s := makeSheet("Item", [][]string{
		{"ID", "Quality"}, {"pk int32", "Quality"}, {"cs", "cs"}, {"", ""}, {"1", "999"},
	})
	root.sheets[s.sheetName] = s
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "does not define numeric value") {
			t.Fatalf("unexpected enum validation result: %v", recovered)
		}
	}()
	s.checks(root)
}

func TestPrimaryKeyUniquenessUsesTypedValue(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"}, {"01"},
	})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "duplicate primary key") {
			t.Fatalf("unexpected typed-primary-key validation result: %v", recovered)
		}
	}()
	s.checkPrimaryKey(NewParser())
}

func TestInvalidExportFilterFailsValidation(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"ID"}, {"pk int32"}, {"client"}, {""}, {"1"},
	})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "invalid export filter") {
			t.Fatalf("unexpected filter validation result: %v", recovered)
		}
	}()
	s.checks(NewParser())
}

func TestForeignKeyComparisonUsesTypedValue(t *testing.T) {
	root := NewParser()
	target := makeSheet("Target", [][]string{
		{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"},
	})
	source := makeSheet("Source", [][]string{
		{"ID", "TargetID"}, {"pk int32", "int32"}, {"cs", "cs"}, {"", ""}, {"1", "01"},
	})
	source.headers[1].tags = []HeadTag{"fk:Target.ID"}
	root.sheets[target.sheetName] = target
	root.sheets[source.sheetName] = source
	source.checkTags(root)
}

func TestDataSheetPrimaryKeyMustReachBothTargets(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"ID"}, {"pk int32"}, {"c"}, {""}, {"1"},
	})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "both client and server") {
			t.Fatalf("unexpected primary-key filter result: %v", recovered)
		}
	}()
	s.checks(NewParser())
}

func TestNestedScalarValuesAreValidated(t *testing.T) {
	root := NewParser()
	child := makeSheet("Child", [][]string{
		{"Value"}, {"int32"}, {"cs"}, {""},
	})
	parent := makeSheet("Parent", [][]string{
		{"ID", "Child"}, {"pk int32", "Child"}, {"cs", "cs"}, {"", ""}, {"1", "invalid"},
	})
	root.sheets[child.sheetName] = child
	root.sheets[parent.sheetName] = parent
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "parse int32 failed") {
			t.Fatalf("unexpected nested scalar validation result: %v", recovered)
		}
	}()
	parent.checkCustomMessageValues(root)
}

func TestI18nKeyCollisionsAreRejected(t *testing.T) {
	root := NewParser()
	first := makeSheet("A_B", [][]string{
		{"ID", "C"}, {"pk string", "i18n"}, {"cs", "cs"}, {"", ""}, {"x", "first"},
	})
	second := makeSheet("A", [][]string{
		{"ID", "B_C"}, {"pk string", "i18n"}, {"cs", "cs"}, {"", ""}, {"x", "second"},
	})
	root.sheets[first.sheetName] = first
	root.sheets[second.sheetName] = second
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "i18n key") {
			t.Fatalf("unexpected i18n collision result: %v", recovered)
		}
	}()
	root.checkI18nKeyCollisions()
}

func TestDataRowsCannotContainUnnamedColumns(t *testing.T) {
	s := NewSheetParser("Item")
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "data row has 2 columns") {
			t.Fatalf("unexpected extra-column result: %v", recovered)
		}
	}()
	s.ParseRows([][]string{{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1", "unexpected"}})
}

func TestGeneratedNamesCannotCollideByCase(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	root.sheets["Item"] = makeSheet("Item", [][]string{{"ID"}, {"pk int32"}, {"cs"}, {""}})
	root.sheets["item"] = makeSheet("item", [][]string{{"ID"}, {"pk int32"}, {"cs"}, {""}})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "case-insensitive") {
			t.Fatalf("unexpected generated-name collision result: %v", recovered)
		}
	}()
	root.checkGeneratedNameCollisions()
}

func TestGeneratedCsharpFilenamesCannotCollide(t *testing.T) {
	configureTestExports(t)
	root := NewParser()
	root.sheets["ItemQuality"] = makeSheet("ItemQuality", [][]string{{"ID"}, {"pk int32"}, {"cs"}, {""}})
	root.sheets["Item_Quality"] = makeSheet("Item_Quality", [][]string{{"ID"}, {"pk int32"}, {"cs"}, {""}})
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "both generate C#") {
			t.Fatalf("unexpected C# filename collision result: %v", recovered)
		}
	}()
	root.checkGeneratedNameCollisions()
}

func TestInvalidScalarValuesPanicWithLocation(t *testing.T) {
	tests := []struct {
		typeName string
		value    string
		want     string
	}{
		{typeName: "int32", value: "oops", want: "parse int32 failed"},
		{typeName: "int64", value: "oops", want: "parse int64 failed"},
		{typeName: "float", value: "oops", want: "parse float failed"},
		{typeName: "double", value: "oops", want: "parse double failed"},
		{typeName: "bool", value: "yes", want: "parse bool failed"},
	}

	for _, test := range tests {
		t.Run(test.typeName, func(t *testing.T) {
			sheet := makeSheet("Item", [][]string{
				{"Value"}, {test.typeName}, {"cs"}, {"value"}, {test.value},
			})
			sheet.SetSourceFile("assets/xls/item.xlsx")
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Fatal("invalid scalar value must panic")
				}
				message := recovered.(string)
				for _, want := range []string{test.want, "file=\"assets/xls/item.xlsx\"", "sheet=\"Item\"", "excel_row=5", "field=\"Value\""} {
					if !strings.Contains(message, want) {
						t.Fatalf("panic %q does not contain %q", message, want)
					}
				}
			}()
			sheet.TypeNameToValue(NewParser(), sheet.GetFiled(0), 0, test.value)
		})
	}
}

func TestEmbeddedForeignKeyUsesEmbeddedFieldColumn(t *testing.T) {
	root := NewParser()
	target := makeSheet("Item", [][]string{
		{"ID", "Code"}, {"pk int32", "unique string"}, {"cs", "cs"}, {"", ""}, {"1", "SWORD"},
	})
	embedded := makeSheet("Reward", [][]string{
		{"ItemCode", "Amount"}, {"string", "int32"}, {"cs", "cs"}, {"", ""},
	})
	source := makeSheet("Drop", [][]string{
		{"ID", "Rewards"}, {"pk int32", "repeated Reward"}, {"cs", "cs"}, {"", ""}, {"1", "SWORD|5"},
	})
	source.headers[1].tags = []HeadTag{"fk:Reward.ItemCode=Item.Code"}
	root.sheets[target.sheetName] = target
	root.sheets[embedded.sheetName] = embedded
	root.sheets[source.sheetName] = source

	var logs bytes.Buffer
	source.logger = slog.New(slog.NewTextHandler(&logs, nil))
	source.checkTags(root)
	if strings.Contains(logs.String(), "foreign key value not found") {
		t.Fatalf("valid embedded foreign key was rejected: %s", logs.String())
	}
}

func TestEmptyScalarStringForeignKeyIsRejected(t *testing.T) {
	root := NewParser()
	target := makeSheet("Target", [][]string{
		{"Code"}, {"pk string"}, {"cs"}, {""}, {"A"},
	})
	source := makeSheet("Source", [][]string{
		{"ID", "TargetCode"}, {"pk int32", "string"}, {"cs", "cs"}, {"", ""}, {"1", ""},
	})
	source.headers[1].tags = []HeadTag{"fk:Target.Code"}
	root.sheets[target.sheetName] = target
	root.sheets[source.sheetName] = source

	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "foreign key value is empty") {
			t.Fatalf("unexpected empty foreign-key result: %v", recovered)
		}
	}()
	source.checkTags(root)
}

func TestRepeatedFieldCannotBeForeignKeyTarget(t *testing.T) {
	root := NewParser()
	target := makeSheet("Target", [][]string{
		{"ID", "Codes"}, {"pk int32", "repeated string"}, {"cs", "cs"}, {"", ""}, {"1", "A;B"},
	})
	source := makeSheet("Source", [][]string{
		{"ID", "TargetCode"}, {"pk int32", "string"}, {"cs", "cs"}, {"", ""}, {"1", "A"},
	})
	source.headers[1].tags = []HeadTag{"fk:Target.Codes"}
	root.sheets[target.sheetName] = target
	root.sheets[source.sheetName] = source

	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "repeated fields cannot be foreign key targets") {
			t.Fatalf("unexpected repeated foreign-key target result: %v", recovered)
		}
	}()
	source.checkTags(root)
}

func TestForeignKeyTargetMustIdentifyOneRow(t *testing.T) {
	for name, target := range map[string]*SheetParser{
		"ordinary field": makeSheet("Target", [][]string{
			{"ID", "Code"}, {"pk int32", "string"}, {"cs", "cs"}, {"", ""}, {"1", "A"},
		}),
		"composite key member": makeSheet("Target", [][]string{
			{"GroupID", "Code"}, {"pk int32", "pk string"}, {"cs", "cs"}, {"", ""}, {"1", "A"},
		}),
	} {
		t.Run(name, func(t *testing.T) {
			root := NewParser()
			source := makeSheet("Source", [][]string{
				{"ID", "TargetCode"}, {"pk int32", "string"}, {"cs", "cs"}, {"", ""}, {"1", "A"},
			})
			source.headers[1].tags = []HeadTag{"fk:Target.Code"}
			root.sheets[target.sheetName] = target
			root.sheets[source.sheetName] = source
			defer func() {
				if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "must be unique") {
					t.Fatalf("unexpected foreign-key target result: %v", recovered)
				}
			}()
			source.checkTags(root)
		})
	}
}

func TestGeneratedLoaderIdentifiersMustRemainStable(t *testing.T) {
	for name, sheet := range map[string]*SheetParser{
		"sheet underscore": makeSheet("Item_Type", [][]string{{"ID"}, {"pk int32"}, {"cs"}, {""}, {"1"}}),
		"field underscore": makeSheet("Item", [][]string{{"Item_ID"}, {"pk int32"}, {"cs"}, {""}, {"1"}}),
		"lowercase field":  makeSheet("Item", [][]string{{"id"}, {"pk int32"}, {"cs"}, {""}, {"1"}}),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "PascalCase") {
					t.Fatalf("unexpected generated-identifier result: %v", recovered)
				}
			}()
			sheet.checkHeaderSchema(NewParser())
		})
	}
}
