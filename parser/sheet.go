package parser

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"log/slog"
	"regexp"
	"strings"
)

var reTrimTag = regexp.MustCompile(`[\n\t\r ]+`)

/*
 * 数据表处理
 */

// 客户端或服务器
type Sheet struct {
	sheetName string

	filter string // c or s

	// 表头
	headers        []Head
	headersIndexes map[string]int32 // key is name, value is index

	// 数据
	data [][]string
}

type SheetParser struct {
	Sheet
	logger *slog.Logger
}

func NewSheetParser(sheetName string) *SheetParser {
	return &SheetParser{
		logger: slog.Default().With(slog.String("ThisSheet", sheetName)),
		Sheet: Sheet{
			sheetName:      sheetName,
			headersIndexes: make(map[string]int32),
		},
	}
}

// GetFieldColIndex 获得指定字段名的列索引
func (s *SheetParser) GetFieldColIndex(name string) int32 {
	return s.headersIndexes[name]
}

func (s *SheetParser) Parse(f *excelize.File) bool {
	rows, _ := f.GetRows(s.sheetName)
	s.parseHeader(rows)
	s.parseHeadTags(f)
	s.parseData(rows)

	return true
}

func (s *SheetParser) isHeader(rowIndex int32) bool {
	return rowIndex < HeadCount
}

func (s *SheetParser) parseHeader(rows [][]string) {
	// 预先分配字段数量
	s.headers = make([]Head, len(rows[0]))

	for i, row := range rows {
		if !s.isHeader(int32(i)) {
			break
		}

		for j, cell := range row {
			s.headers[j].info[i] = cell
		}
	}

	// build indexes
	for colIndex, head := range s.headers {
		s.headersIndexes[head.Name()] = int32(colIndex)
	}
}

func (s *SheetParser) parseHeadTags(f *excelize.File) {
	comments, _ := f.GetComments(s.sheetName)
	for _, v := range comments {
		col, row, _ := excelize.CellNameToCoordinates(v.Cell)
		if row != 1 {
			// 出需要处理第一行的批注
			continue
		}

		// 处理批注
		txtTag := v.Paragraph[0].Text
		txtTag = reTrimTag.ReplaceAllString(txtTag, "")

		// 处理空格
		var tags []HeadTag
		ss := strings.Split(txtTag, ";")
		for _, tag := range ss {
			if len(tag) > 0 {
				tags = append(tags, HeadTag(strings.Trim(tag, " ")))
			}
		}

		// 保存标签
		s.headers[col-1].tags = tags
	}
}

func (s *SheetParser) parseData(rows [][]string) {
	for i, row := range rows {
		// 跳过表头
		if s.isHeader(int32(i)) {
			continue
		}

		if len(row) == 0 {
			// 跳过空行
			continue
		}

		s.data = append(s.data, row)
	}
}

// 返回
func (s *SheetParser) checkFiledValueIsUnique(filedName string) (value string, ok bool) {
	// 检查指定列的数据，是否存在重复值
	colIndex := s.GetFieldColIndex(filedName)
	values := make(map[string]bool)
	for _, row := range s.data {
		v := row[colIndex]
		if values[v] {
			return v, false
		}
		values[v] = true
	}

	return "", true
}

// TODO 优化这个方法
// 可以缓存成map，减少重复计算
func (s *SheetParser) hasExistFiledValue(filedName string, filedValue string) bool {
	// 检查指定列的数据，是否存在
	colIndex := s.GetFieldColIndex(filedName)
	for _, row := range s.data {
		value := row[colIndex]
		if value == filedValue {
			return true
		}
	}
	return false
}

func (s *SheetParser) getFiledValue(filedName string, rowIndex int32) string {
	colIndex := s.GetFieldColIndex(filedName)
	return s.data[rowIndex][colIndex]
}

func (s *SheetParser) checks(root *Parser) {
	s.checkPrimaryKey(root)
	s.checkUnique(root)
	s.checkTags(root)
}

// TODO 当前主键只支持一个字段，需要扩展到多个字段
func (s *SheetParser) checkPrimaryKey(root *Parser) {
	for _, head := range s.headers {
		// 当前表的数据唯一性检测
		if head.IsPrimaryKey() {
			value, ok := s.checkFiledValueIsUnique(head.Name())
			if !ok {
				panic(fmt.Sprintf("[%v.%v]check pk fail, has same value %v",
					s.sheetName, head.Name(), value))
			}
		}
	}
}

// checkUnique 当前表的数据唯一性检测
func (s *SheetParser) checkUnique(root *Parser) {
	for _, head := range s.headers {
		if head.IsUnique() {
			value, ok := s.checkFiledValueIsUnique(head.Name())
			if !ok {
				panic(fmt.Sprintf("[%v.%v]check unique fail, has same value %v",
					s.sheetName, head.Name(), value))
			}
		}
	}
}

// checkTags 根据tag处理检查
func (s *SheetParser) checkTags(root *Parser) {
	for _, head := range s.headers {
		for _, tag := range head.tags {
			// 先检测外键
			key := tag.GetKey()
			switch key {
			case TagFkName:
				embedSheetName, embedFiledName, fkSheetName, fkFiledName := tag.ParseForeignKey()
				if len(embedSheetName) == 0 {
					// 没有内嵌结构

					// 检测外键的表是否存在
					if !root.hasSheetParser(fkSheetName) {
						panic(fmt.Sprintf("[%v.%v]check foreign key fail, sheet not found: %v", s.sheetName, head.Name(), fkSheetName))
					}

					// 检测外键的字段值是否存在
					refSheetParser := root.getSheetParser(fkSheetName)
					for rowIndex := range s.data {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						if !refSheetParser.hasExistFiledValue(fkFiledName, thisValue) {
							panic(fmt.Sprintf("[%v.%v]check foreign key fail, ref value is excel row=%v, value=%v, not found: %v.%v",
								s.sheetName, head.Name(),
								DataRowIndex2ExcelRow(int32(rowIndex)), thisValue, fkSheetName, fkFiledName))
						}
					}

				} else {
					// 有内嵌结构
					// TODO
					_ = embedSheetName
					_ = embedFiledName
					embedSheetName = embedSheetName
					slog.Warn("not implemented tag checker", "tag", tag)
				}

			case TagIndexName:
				// TODO
			}
		}
	}
}
