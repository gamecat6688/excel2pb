package parser

import (
	"excel2pb/config"
	"excel2pb/works"
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/xuri/excelize/v2"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Parser struct {
	sheets map[string]*SheetParser
	enums  map[string]*EnumParser
	sync.Map
	sync.RWMutex
}

func NewParser() *Parser {
	return &Parser{
		sheets: make(map[string]*SheetParser),
		enums:  make(map[string]*EnumParser),
	}
}

func (p *Parser) ParseExcels() {
	slog.Info("parse excels start")

	// 读取所有文件夹(包含子文件夹)下的excel文件，排除~开头的临时文件
	matches, err := doublestar.FilepathGlob(fmt.Sprintf("%v/**/[!~]*.xlsx", config.Cfg.ExcelDir))
	if err != nil {
		slog.Error("ParseExcels fail", "error", err)
		return
	}

	// 并发解析excel
	for _, fileName := range matches {
		works.Go(func() {
			slog.Info("parsing excel", "fileName", fileName)
			p.parseFromFile(fileName)
		})
	}
	works.Wait()

	slog.Info("parse excels over")
}

func (p *Parser) parseFromFile(excelFile string) {
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
				slog.Error("parseFromFile sheet fail", "sheetName", sheetName)
				return
			}

			p.Lock()
			p.enums[enumName] = ps
			p.Unlock()
		} else {
			// 数据表
			if p.hasSheetParser(sheetName) {
				slog.Error("sheet already exist", "sheetName", sheetName)
				return
			}

			ps := NewSheetParser(sheetName)
			rows, _ := f.GetRows(sheetName)
			ps.ParseRows(rows)
			ps.ParseHeadTags(f)

			p.Lock()
			p.sheets[sheetName] = ps
			p.Unlock()
		}
	}
}

func (p *Parser) GetI18nFilePath() string {
	return fmt.Sprintf("%v/%v.xlsx", config.Cfg.ExcelDir, I18nSheetName)
}

func (p *Parser) MergeI18n() {
	if !config.Cfg.EnableI18n {
		return
	}

	slog.Info("merge i18n start")

	//defer TimeCost("MergeI18n")()

	var i18n *I18nParser
	f, err := excelize.OpenFile(p.GetI18nFilePath())
	if err == nil {
		defer f.Close()

		// 读取已存在的i18n文件
		ps := NewSheetParser(I18nSheetName)
		rows, _ := f.GetRows(I18nSheetName)
		ps.ParseRows(rows)

		i18n = NewI18nParser(ps)
	} else {
		// 创建新的i18n文件
		i18n = NewI18nParser(nil)
	}

	// 合并新的多语言数据
	for _, v := range p.sheets {
		i18n.MergeSheet(v)
	}

	i18n.WriteToExcel(p.GetI18nFilePath())

	p.Lock()
	p.sheets[I18nSheetName] = i18n.SheetParser
	p.Unlock()

	slog.Info("merge i18n end")
}

func (p *Parser) Export() {
	p.checks()
	p.exportProto()
	p.exportData()
	p.exportCode()
	p.exportPb()
}

func (p *Parser) hasSheetParser(sheetName string) bool {
	p.RLock()
	defer p.RUnlock()

	_, ok := p.sheets[sheetName]
	return ok
}

func (p *Parser) getSheetParser(sheetName string) *SheetParser {
	p.RLock()
	defer p.RUnlock()

	return p.sheets[sheetName]
}

func (p *Parser) hasEnumParser(sheetName string) bool {
	p.RLock()
	defer p.RUnlock()

	_, ok := p.enums[sheetName]
	return ok
}

func (p *Parser) getEnumParser(sheetName string) *EnumParser {
	p.RLock()
	defer p.RUnlock()

	return p.enums[sheetName]
}

func (p *Parser) checks() {
	for _, sheet := range p.sheets {
		works.Go(func() {
			slog.Info("check excels", "sheetName", sheet.sheetName)
			sheet.checks(p)
		})
	}
	works.Wait()
}

func (p *Parser) exportProto() {
	for _, v := range p.enums {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export enum proto", "sheetName", v.sheetName, "filter", f)
				v.ExportProto(f)
			})
		}
	}

	for _, v := range p.sheets {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export message proto", "sheetName", v.sheetName, "filter", f)
				ns := v.SplitByFilter(f)
				ns.ExportProto(p)
			})
		}
	}

	works.Wait()
}

// exportPb 导出pb
func (p *Parser) exportPb() {
	for _, filter := range AllFilters {
		cfg := config.Cfg.Outs[FilterFullName[filter]]

		protoOutPath := cfg.ProtoPath
		pbOutPath := cfg.PbPath

		// make dirs
		os.MkdirAll(pbOutPath, os.ModePerm)

		sss, err := filepath.Glob(fmt.Sprintf("%v/*.proto", protoOutPath))
		if err != nil {
			slog.Error("export pb fail", "error", err)
			return
		}

		for _, filename := range sss {
			slog.Info("export pb file", "filename", filename)

			argInclude := fmt.Sprintf("--proto_path=%v", protoOutPath)
			argOut := ""

			switch cfg.CodeLanguage {
			case "golang":
				argOut = fmt.Sprintf("--go_out=%v", pbOutPath)
			case "csharp":
				argOut = fmt.Sprintf("--csharp_out=%v", pbOutPath)
			case "java":
				argOut = fmt.Sprintf("--java_out=%v", pbOutPath)
			case "cpp":
				argOut = fmt.Sprintf("--cpp_out=%v", pbOutPath)
			}

			works.Go(func() {
				cmd := exec.Command("protoc", argInclude, argOut, filename)
				err = cmd.Run()
				if err != nil {
					slog.Error("protoc failed", "error", err, "filename", filename)
					//log.Fatalf("protoc failed: %v", err)
				}
			})
		}
	}

	works.Wait()
}

func (p *Parser) exportData() {
	for _, v := range p.sheets {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export bin file", "sheetName", v.sheetName, "filter", f)
				ns := v.SplitByFilter(f)
				ns.ExportData(p)
			})
		}
	}

	works.Wait()
}

func (p *Parser) exportCode() {
	p.exportLoadCode()
	p.exportModuleCode()
}

func (p *Parser) exportLoadCode() {
	for _, f := range AllFilters {
		slog.Info("export load code", "filter", f)

		cfg := config.Cfg.Outs[FilterFullName[f]]
		tplCodePath := config.Cfg.TplCodePaths[cfg.CodeLanguage]
		codeOutPath := config.Cfg.CodeOutPaths[cfg.CodeLanguage]
		switch cfg.CodeLanguage {
		case "golang":
			moduleCode := NewGolangLoaderCode(tplCodePath, codeOutPath)
			moduleCode.GenCode(p)
		case "csharp":
			moduleCode := NewCsharpLoaderCode(tplCodePath, codeOutPath)
			moduleCode.GenCode(p)
		}
	}
}

func (p *Parser) exportModuleCode() {
	for _, v := range p.sheets {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export module code", "sheetName", v.sheetName, "filter", f)
				ns := v.SplitByFilter(f)
				ns.ExportCode(p)
			})
		}
	}

	works.Wait()
}
