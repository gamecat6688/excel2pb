package parser

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseHeadTagsSkipsEmptyCommentParagraph(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	parser := NewSheetParser("Sheet1")
	parser.ParseRows([][]string{{"Id"}, {"pk string"}, {"cs"}, {"identifier"}})
	if err := f.AddComment("Sheet1", excelize.Comment{Cell: "A1"}); err != nil {
		t.Fatalf("add empty comment: %v", err)
	}

	parser.ParseHeadTags(f)
	if len(parser.headers[0].tags) != 0 {
		t.Fatalf("empty comment must not create tags: %#v", parser.headers[0].tags)
	}
}

func TestCheckCustomFieldCountReportsConfigLocation(t *testing.T) {
	parser := NewSheetParser("DropPool")
	parser.SetSourceFile("assets/xls/drop_pool.xlsx")
	child := NewSheetParser("DropEntry")
	child.ParseRows([][]string{{"ItemId", "Weight", "MinQuantity", "MaxQuantity"}, {"int32", "int32", "int32", "int32"}, {"cs", "cs", "cs", "cs"}, {"", "", "", ""}})

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected invalid composite value panic")
		}
		message := recovered.(string)
		for _, fragment := range []string{"file=\"assets/xls/drop_pool.xlsx\"", "sheet=\"DropPool\"", "excel_row=5", "field=\"Entries\"", "DropEntry expects 4 fields", "got 2"} {
			if !strings.Contains(message, fragment) {
				t.Fatalf("panic %q does not contain %q", message, fragment)
			}
		}
	}()

	parser.checkCustomFieldCount(Head{info: [HeadCount]string{HeadName: "Entries", HeadType: "repeated DropEntry"}}, 0, "10039|60", []string{"10039", "60"}, child)
}
