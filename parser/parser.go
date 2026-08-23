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
		slog.Error("find Excel workbooks failed", "excel_dir", config.Cfg.ExcelDir, "error", err)
		return
	}
	if len(matches) == 0 {
		slog.Warn("no Excel workbooks found", "excel_dir", config.Cfg.ExcelDir)
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
		slog.Error("open Excel workbook failed", "file", excelFile, "error", err)
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
				slog.Error("duplicate enum sheet name", "file", excelFile, "sheet", sheetName)
				return
			}

			enumName := SplitEnumName(sheetName)
			ps := NewEnumParser(sheetName)
			ps.SetSourceFile(excelFile)
			if !ps.Parse(f) {
				slog.Error("parse enum sheet failed", "file", excelFile, "sheet", sheetName)
				return
			}

			p.Lock()
			p.enums[enumName] = ps
			p.Unlock()
		} else {
			// 数据表
			if p.hasSheetParser(sheetName) {
				slog.Error("duplicate data sheet name", "file", excelFile, "sheet", sheetName)
				return
			}

			ps := NewSheetParser(sheetName)
			ps.SetSourceFile(excelFile)
			rows, err := f.GetRows(sheetName)
			if err != nil {
				slog.Error("read data sheet rows failed", "file", excelFile, "sheet", sheetName, "error", err)
				continue
			}
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
	i18nPath := p.GetI18nFilePath()

	//defer TimeCost("MergeI18n")()

	var i18n *I18nParser
	f, err := excelize.OpenFile(i18nPath)
	if err == nil {
		defer f.Close()

		// 读取已存在的i18n文件
		ps := NewSheetParser(I18nSheetName)
		ps.SetSourceFile(i18nPath)
		rows, err := f.GetRows(I18nSheetName)
		if err != nil {
			slog.Error("read I18N sheet rows failed", "file", i18nPath, "sheet", I18nSheetName, "error", err)
			return
		}
		ps.ParseRows(rows)

		i18n = NewI18nParser(ps)
	} else {
		// 创建新的i18n文件
		i18n = NewI18nParser(nil)
		i18n.SetSourceFile(i18nPath)
	}

	// 合并新的多语言数据
	for _, v := range p.sheets {
		i18n.MergeSheet(v)
	}

	if !i18n.WriteToExcel(i18nPath) {
		slog.Error("write merged I18N workbook failed", "file", i18nPath)
		return
	}

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
			slog.Info("check Excel sheet", "sheet", sheet.sheetName, "source_file", sheet.sourceFile)
			sheet.checks(p)
		})
	}
	works.Wait()
}

func (p *Parser) exportProto() {
	for _, v := range p.enums {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export enum proto", "sheet", v.sheetName, "source_file", v.sourceFile, "filter", f)
				v.ExportProto(f)
			})
		}
	}

	for _, v := range p.sheets {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export message proto", "sheet", v.sheetName, "source_file", v.sourceFile, "filter", f)
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
		if err := os.MkdirAll(pbOutPath, os.ModePerm); err != nil {
			slog.Error("create protobuf code output directory failed", "filter", filter, "output_path", pbOutPath, "error", err)
			continue
		}

		sss, err := filepath.Glob(fmt.Sprintf("%v/*.proto", protoOutPath))
		if err != nil {
			slog.Error("find generated proto files failed", "filter", filter, "proto_path", protoOutPath, "error", err)
			return
		}

		for _, filename := range sss {
			slog.Info("export pb file", "filename", filename)

			argInclude := fmt.Sprintf("--proto_path=%v", protoOutPath)
			argOut, supported := protobufOutArg(cfg.CodeLanguage, pbOutPath)
			if !supported {
				slog.Error("unsupported protobuf code language", "filter", filter, "code_language", cfg.CodeLanguage, "proto_file", filename)
				continue
			}

			works.Go(func() {
				cmd := exec.Command("protoc", argInclude, argOut, filename)
				output, err := cmd.CombinedOutput()
				if err != nil {
					slog.Error("protoc failed", "filter", filter, "code_language", cfg.CodeLanguage, "proto_file", filename, "proto_path", protoOutPath, "output_path", pbOutPath, "error", err, "output", strings.TrimSpace(string(output)))
				}
			})
		}
	}

	works.Wait()
}

func protobufOutArg(codeLanguage, outputPath string) (string, bool) {
	flags := map[string]string{
		"golang": "go",
		"csharp": "csharp",
		"godot":  "gdscript",
		"java":   "java",
		"cpp":    "cpp",
	}
	flag, ok := flags[codeLanguage]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("--%s_out=%v", flag, outputPath), true
}

func (p *Parser) exportData() {
	for _, v := range p.sheets {
		for _, f := range AllFilters {
			works.Go(func() {
				slog.Info("export binary data", "sheet", v.sheetName, "source_file", v.sourceFile, "filter", f)
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
		case "godot":
			moduleCode := NewGodotLoaderCode(tplCodePath, codeOutPath)
			moduleCode.GenCode(p)
		default:
			slog.Error("unsupported load code language", "filter", f, "code_language", cfg.CodeLanguage)
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
