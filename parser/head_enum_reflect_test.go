package parser

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestHeadTagsAndTypeClassification(t *testing.T) {
	plain := HeadTag("fk:Item.ID")
	if !plain.IsForeignKey() || plain.GetKey() != TagFkName {
		t.Fatal("plain foreign-key tag was not recognized")
	}
	embedSheet, embedField, fkSheet, fkField, err := HeadTag("fk:Reward.ItemID=Item.ID").ParseForeignKey()
	if err != nil {
		t.Fatal(err)
	}
	if embedSheet != "Reward" || embedField != "ItemID" || fkSheet != "Item" || fkField != "ID" {
		t.Fatalf("unexpected embedded foreign key: %q.%q -> %q.%q", embedSheet, embedField, fkSheet, fkField)
	}

	h := Head{info: [HeadCount]string{HeadType: "repeated int32", HeadExport: "cs"}}
	if h.BaseType() != "int32" || h.ProtoType() != "repeated int32" || !h.IsRepeated() || !h.IsExportClient() || !h.IsExportServer() {
		t.Fatalf("unexpected repeated head classification: %#v", h)
	}
	pk := Head{info: [HeadCount]string{HeadType: "pk int32"}}
	unique := Head{info: [HeadCount]string{HeadType: "unique string"}}
	i18n := Head{info: [HeadCount]string{HeadType: "i18n"}}
	if !pk.IsPrimaryKey() || !unique.IsUnique() || !i18n.IsI18n() {
		t.Fatal("field modifiers were not recognized")
	}
	timestamp := Head{info: [HeadCount]string{HeadType: TimestampName}}
	if timestamp.ProtoType() != "int64" {
		t.Fatal("timestamp must export as int64")
	}
	repeatedTimestamp := Head{info: [HeadCount]string{HeadType: "repeated timestamp"}}
	if repeatedTimestamp.ProtoType() != "repeated int64" {
		t.Fatalf("repeated timestamp proto type = %q", repeatedTimestamp.ProtoType())
	}
}

func TestMalformedForeignKeyTagReturnsError(t *testing.T) {
	if _, _, _, _, err := HeadTag("fk:MissingDot").ParseForeignKey(); err == nil {
		t.Fatal("malformed foreign key tag must return an error")
	}
}

func TestEnumRegistrationIsAtomic(t *testing.T) {
	root := NewParser()
	var registered atomic.Int32
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if root.addEnumParser("Item_Quality", &EnumParser{name: "Item_Quality"}) {
				registered.Add(1)
			}
		}()
	}
	wait.Wait()
	if registered.Load() != 1 {
		t.Fatalf("enum registered %d times, want exactly once", registered.Load())
	}
}

func TestEnumParser(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetRow("Sheet1", "A1", &[]interface{}{"Name", "Value", "Comment"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Sheet1", "A2", &[]interface{}{"None", "0", "default"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Sheet1", "A3", &[]interface{}{"Rare", "2"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetName("Sheet1", "Quality_Enum"); err != nil {
		t.Fatal(err)
	}
	enum := NewEnumParser("Quality_Enum")
	enum.SetSourceFile("assets/xls/quality.xlsx")
	if !enum.Parse(f) || enum.name != "Quality" || len(enum.enums) != 2 || enum.getEnumValue("Rare") != 2 {
		t.Fatalf("unexpected enum parser result: %#v", enum)
	}
	if enum.sourceFile != "assets/xls/quality.xlsx" {
		t.Fatalf("enum source file = %q", enum.sourceFile)
	}

	if _, err := f.NewSheet("Broken_Enum"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Broken_Enum", "A1", &[]interface{}{"Name", "Value"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Broken_Enum", "A2", &[]interface{}{"OnlyName"}); err != nil {
		t.Fatal(err)
	}
	if NewEnumParser("Broken_Enum").Parse(f) {
		t.Fatal("enum row without a value must be rejected")
	}

}

func TestEnumParserRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name string
		rows [][]interface{}
	}{
		{name: "InvalidValue_Enum", rows: [][]interface{}{{"Name", "Value"}, {"Invalid", "not-a-number"}}},
		{name: "NonZero_Enum", rows: [][]interface{}{{"Name", "Value"}, {"First", 1}}},
		{name: "DuplicateName_Enum", rows: [][]interface{}{{"Name", "Value"}, {"None", 0}, {"None", 1}}},
		{name: "DuplicateValue_Enum", rows: [][]interface{}{{"Name", "Value"}, {"None", 0}, {"Other", 0}}},
		{name: "ReservedName_Enum", rows: [][]interface{}{{"Name", "Value"}, {"message", 0}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := excelize.NewFile()
			defer func() { _ = file.Close() }()
			if err := file.SetSheetName("Sheet1", test.name); err != nil {
				t.Fatal(err)
			}
			for rowIndex, row := range test.rows {
				cell, err := excelize.CoordinatesToCellName(1, rowIndex+1)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.SetSheetRow(test.name, cell, &row); err != nil {
					t.Fatal(err)
				}
			}
			if NewEnumParser(test.name).Parse(file) {
				t.Fatal("invalid enum definition must be rejected")
			}
		})
	}
}

func TestSetDynamicFields(t *testing.T) {
	files, err := parseProtoFile([]string{"record.proto"}, []string{"testdata"})
	if err != nil {
		t.Fatalf("parse test proto: %v", err)
	}
	messageType, err := getMessageType(files, "test.Record")
	if err != nil {
		t.Fatalf("get message type: %v", err)
	}
	msg := messageType.New()
	setDynamicFields(msg, map[string]interface{}{
		"title": "record",
		"ids":   []interface{}{int32(4), int32(8)},
		"child": map[string]interface{}{"name": "nested"},
	})
	fields := msg.Descriptor().Fields()
	if msg.Get(fields.ByName("title")).String() != "record" || msg.Get(fields.ByName("ids")).List().Len() != 2 || msg.Get(fields.ByName("child")).Message().Get(fields.ByName("child").Message().Fields().ByName("name")).String() != "nested" {
		t.Fatalf("dynamic message fields were not set: %v", msg)
	}
}
