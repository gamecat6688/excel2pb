package parser

import (
	"excel2pb/config"
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
)

type Parser struct {
	sheets map[string]*SheetParser
	enums  map[string]*EnumParser
}

func NewParser() *Parser {
	return &Parser{
		sheets: make(map[string]*SheetParser),
		enums:  make(map[string]*EnumParser),
	}
}

func (p *Parser) ParseExcels() {
	sss, err := filepath.Glob(fmt.Sprintf("%v/*.xlsx", config.ExcelDir))
	if err != nil {
		log.Println("err:", err)
		return
	}

	// 遍历并过滤掉临时文件
	var excels []string
	for _, f := range sss {
		if !strings.HasPrefix(filepath.Base(f), "~") {
			excels = append(excels, f)
		}
	}

	// 并发解析excel

	var wg sync.WaitGroup
	for _, f := range excels {
		wg.Add(1)
		go func(fileName string) {
			p.parse(fileName)
			wg.Done()
		}(f)
	}
	wg.Wait()
}

func (p *Parser) parse(excelFile string) {
	f, err := excelize.OpenFile(excelFile)
	if err != nil {
		slog.Error("open file fail", "error", err)
		return
	}
	defer f.Close()

	for _, sheetName := range f.GetSheetList() {
		if strings.HasPrefix(sheetName, "#") {
			// 跳过#开头的表格,备注类辅助表
			continue
		}

		if strings.HasSuffix(sheetName, "Enum") {
			// 枚举表
			if p.hasEnumParser(sheetName) {
				slog.Error("sheet already exist", "sheetName", sheetName)
				return
			}

			enumName := SplitEnumName(sheetName)
			ps := NewEnumParser(sheetName)
			if !ps.Parse(f) {
				slog.Error("parse sheet fail", "sheetName", sheetName)
				return
			}

			p.enums[enumName] = ps
		} else {
			// 数据表
			if p.hasSheetParser(sheetName) {
				slog.Error("sheet already exist", "sheetName", sheetName)
				return
			}

			ps := NewSheetParser(sheetName)
			if !ps.Parse(f) {
				slog.Error("parse sheet fail", "sheetName", sheetName)
				return
			}

			p.sheets[sheetName] = ps
		}
	}
}

func (p *Parser) Export() {
	p.checks()
	p.exportProto()
	p.exportPb()
	p.exportCode()
	p.exportData()
}

func (p *Parser) hasSheetParser(sheetName string) bool {
	_, ok := p.sheets[sheetName]
	return ok
}

func (p *Parser) getSheetParser(sheetName string) *SheetParser {
	return p.sheets[sheetName]
}

func (p *Parser) hasEnumParser(sheetName string) bool {
	_, ok := p.enums[sheetName]
	return ok
}

func (p *Parser) checks() {
	for _, sheet := range p.sheets {
		sheet.checks(p)
	}
}

func (p *Parser) exportProto() {
	for _, v := range p.sheets {
		for _, f := range AllFilters {
			ns := v.SplitByFilter(f)
			ns.ExportProto(p)
		}
	}
}

// exportPb 导出pb
func (p *Parser) exportPb() {
	for _, v := range p.sheets {
		for _, f := range AllFilters {
			ns := v.SplitByFilter(f)
			ns.ExportPb(p)
		}
	}
}

func (p *Parser) exportCode() {
}

func (p *Parser) exportData() {
	for _, v := range p.sheets {
		for _, f := range AllFilters {
			ns := v.SplitByFilter(f)
			ns.ExportData(p)
		}
	}
}
