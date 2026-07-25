package parser

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestHeadTagsAndTypeClassification(t *testing.T) {
	plain := HeadTag("fk:Item.ID")
	if !plain.IsForeignKey() || plain.GetKey() != TagFkName {
		t.Fatal("plain foreign-key tag was not recognized")
	}
	embedSheet, embedField, fkSheet, fkField := HeadTag("fk:Reward.ItemID=Item.ID").ParseForeignKey()
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
	if !enum.Parse(f) || enum.name != "Quality" || len(enum.enums) != 2 || enum.getEnumValue("Rare") != 2 {
		t.Fatalf("unexpected enum parser result: %#v", enum)
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
