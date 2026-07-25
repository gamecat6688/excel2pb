package parser

import (
	"github.com/xuri/excelize/v2"
	"log/slog"
	"sort"
	"strings"
)

const (
	I18nSheetName = "I18N"
)

func NewI18nParser(sheet *SheetParser) *I18nParser {
	rv := &I18nParser{}
	if sheet != nil {
		rv.SheetParser = sheet
	} else {
		rv.SheetParser = NewSheetParser(I18nSheetName)
		rv.initHeaders()
	}
	rv.filter = ClientFlag + ServerFlag
	return rv
}

type I18nParser struct {
	*SheetParser
	indexes map[string]int
}

func (i *I18nParser) initHeaders() {
	var rows [][]string
	rows = append(rows, []string{"ID", "Cn", "Tn", "En"})
	rows = append(rows, []string{"pk string", "string", "string", "string"})
	rows = append(rows, []string{"cs", "cs", "cs", "cs"})
	rows = append(rows, []string{"编号", "简体中文", "繁体中文", "英语"})
	i.parseHeader(rows)
}

func (i *I18nParser) GetName() string {
	return i.sheetName
}

func (i *I18nParser) tryMakeIndex() {
	if i.indexes == nil {
		i.indexes = map[string]int{}

		for r := 0; r < len(i.dataRows); r++ {
			id := i.dataRows[r][0]
			i.indexes[id] = r
		}
	}
}

func (i *I18nParser) getIndex(id string) int {
	index, ok := i.indexes[id]
	if !ok {
		return -1
	}

	return index
}

func (i *I18nParser) MergeSheet(s *SheetParser) {
	for rowIdx, row := range s.dataRows {
		for colIdx, value := range row {
			fd := s.GetFiled(int32(colIdx))
			if fd.IsI18n() && value != "" {
				pkValue := s.GetI18nPrimaryKey(int32(rowIdx))
				i18nKey := MakeI18nKey(s.sheetName, fd.Name(), pkValue)
				i.SetData(i18nKey, value)
			}
		}
	}
}

func (i *I18nParser) SetData(id string, cn string) {
	i.tryMakeIndex()

	// 数据
	index := i.getIndex(id)
	if index == -1 {
		lineData := make([]string, len(i.headers))
		lineData[0] = id
		lineData[1] = cn
		i.dataRows = append(i.dataRows, lineData)
		i.indexes[id] = len(i.dataRows) - 1
	} else {
		i.dataRows[index][0] = id
		i.dataRows[index][1] = cn
	}
}

func (i *I18nParser) sortDataRows() {
	// 排序
	sort.SliceStable(i.dataRows, func(m, n int) bool {
		ss1 := strings.Split(i.dataRows[m][0], "_")
		par1 := ss1[0] + ss1[1]
		id1 := ss1[2]

		ss2 := strings.Split(i.dataRows[n][0], "_")
		par2 := ss2[0] + ss2[1]
		id2 := ss2[2]

		// 先比较首字母
		if par1 != par2 {
			return par1 < par2
		}

		if IsNumber(id1) {
			// 数字比较
			return ToInt64(id1) < ToInt64(id2)
		}

		// 字符比较
		return id1 < id2
	})
}

func (i *I18nParser) WriteToExcel(excelFile string) bool {
	i.sortDataRows()

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("create i18n file fail", "error", err)
		}
	}()

	f.NewSheet(I18nSheetName)
	f.DeleteSheet("Sheet1")

	// write header rows
	for idx, row := range i.headRows {
		cell, err := excelize.CoordinatesToCellName(1, idx+1)
		if err != nil {
			slog.Error("CoordinatesToCellName fail", "error", err)
			return false
		}

		f.SetSheetRow(I18nSheetName, cell, &row)
	}

	// write dataRows rows
	headerRowCount := len(i.headRows)
	for idx, row := range i.dataRows {
		cell, err := excelize.CoordinatesToCellName(1, headerRowCount+idx+1)
		if err != nil {
			slog.Error("CoordinatesToCellName fail", "error", err)
			return false
		}
		f.SetSheetRow(I18nSheetName, cell, &row)
	}

	// 根据指定路径保存文件
	if err := f.SaveAs(excelFile); err != nil {
		slog.Error("save I18n file fail", "error", err)
		return false
	}

	return true
}

//func (i *i18n) WriteToExcel(excelFile string) bool {
//	var file *xlsx.File
//	var sheet *xlsx.Sheet
//	var row *xlsx.Row
//	var cell *xlsx.Cell
//	var err error
//
//	file = xlsx.NewFile()
//	sheet, err = file.AddSheet("通用")
//	if err != nil {
//		fmt.Printf(err.Error())
//		return false
//	}
//
//	// 变量名
//	row = sheet.AddRow()
//	for _, v := range i.headerName {
//		cell = row.AddCell()
//		cell.Value = v
//	}
//
//	// 变量名
//	row = sheet.AddRow()
//	for _, v := range i.headerType {
//		cell = row.AddCell()
//		cell.Value = v
//	}
//
//	// 导出类型
//	row = sheet.AddRow()
//	for _, v := range i.headerCS {
//		cell = row.AddCell()
//		cell.Value = v
//	}
//
//	// 描述
//	row = sheet.AddRow()
//	for _, v := range i.headerDesc {
//		cell = row.AddCell()
//		cell.Value = v
//	}
//
//	// 排序
//	sort.SliceStable(i.dataList, func(m, n int) bool {
//		ss1 := strings.SplitAfter(i.dataList[m][0], "_")
//		par1 := ss1[0] + ss1[1]
//		id1 := ss1[2]
//
//		ss2 := strings.SplitAfter(i.dataList[n][0], "_")
//		par2 := ss2[0] + ss2[1]
//		id2 := ss2[2]
//
//		// 先比较首字母
//		if id1 != id2 {
//			return par1 < par2
//		}
//
//		if isNumber(id1) {
//			// 数字比较
//			return ToUint(id1) < ToUint(id2)
//		}
//
//		// 字符比较
//		return id1 < id2
//	})
//
//	// dataRows
//	nRow := len(i.dataList)
//	for r := 0; r < nRow; r++ {
//		row = sheet.AddRow()
//		for c := 0; c < len(i.headerName); c++ {
//			cell = row.AddCell()
//			cell.Value = i.dataList[r][c]
//		}
//	}
//
//	err = file.SetData(excelFile)
//	if err != nil {
//		fmt.Printf(err.Error())
//	}
//
//	return true
//}
