package parser

import (
	"strings"
	"testing"
)

func makeSheet(name string, rows [][]string) *SheetParser {
	s := NewSheetParser(name)
	s.ParseRows(rows)
	return s
}

func TestSheetParsingFilteringAndPrimaryKeys(t *testing.T) {
	s := makeSheet("Item", [][]string{
		{"ID", "Name", "ServerOnly"},
		{"pk int32", "string", "string"},
		{"cs", "c", "s"},
		{"identifier", "name", "server data"},
		{"1", "Sword", "secret"},
	})

	if s.GetFiled(0).BaseType() != "int32" || !s.GetFiled(0).IsPrimaryKey() {
		t.Fatalf("unexpected primary key header: %#v", s.GetFiled(0))
	}
	client := s.SplitByFilter(ClientFlag)
	if len(client.headers) != 2 || len(client.dataRows[0]) != 2 || client.dataRows[0][1] != "Sword" {
		t.Fatalf("unexpected client sheet: %#v", client)
	}
	server := s.SplitByFilter(ServerFlag)
	if len(server.headers) != 2 || server.dataRows[0][1] != "secret" {
		t.Fatalf("unexpected server sheet: %#v", server)
	}
	if got := s.GetI18nPrimaryKey(0); got != "1" {
		t.Fatalf("GetI18nPrimaryKey = %q", got)
	}
}

func TestCompositePrimaryKeyValidation(t *testing.T) {
	newSheet := func(rows [][]string) *SheetParser {
		return makeSheet("Drop", append([][]string{
			{"ItemID", "Level"},
			{"pk int32", "pk int32"},
			{"cs", "cs"},
			{"", ""},
		}, rows...))
	}

	newSheet([][]string{{"1", "1"}, {"1", "2"}}).checkPrimaryKey(NewParser())

	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(recovered.(string), "duplicate primary key (ItemID,Level)=(1,1)") || !strings.Contains(recovered.(string), "excel_row=6") {
			t.Fatalf("unexpected duplicate primary key error: %v", recovered)
		}
	}()
	newSheet([][]string{{"1", "1"}, {"1", "1"}}).checkPrimaryKey(NewParser())
}

func TestI18nSetDataMergeAndSort(t *testing.T) {
	i18n := NewI18nParser(nil)
	i18n.SetData("Item_Name_10", "old")
	i18n.SetData("Item_Name_10", "new")
	i18n.SetData("Armor_Name_2", "armor")
	i18n.SetData("Item_Name_2", "item")
	if len(i18n.dataRows) != 3 || i18n.dataRows[0][1] != "new" {
		t.Fatalf("SetData must update existing row: %#v", i18n.dataRows)
	}

	i18n.sortDataRows()
	got := []string{i18n.dataRows[0][0], i18n.dataRows[1][0], i18n.dataRows[2][0]}
	want := []string{"Armor_Name_2", "Item_Name_2", "Item_Name_10"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("sorted keys = %#v, want %#v", got, want)
	}
}
