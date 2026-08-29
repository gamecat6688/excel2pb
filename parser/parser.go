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
	"sort"
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
		panic(fmt.Sprintf("find Excel workbooks in %q failed: %v", config.Cfg.ExcelDir, err))
	}
	i18nPath := p.GetI18nFilePath()
	sourceMatches := matches[:0]
	for _, filename := range matches {
		if sameExistingFile(filename, i18nPath) {
			continue
		}
		sourceMatches = append(sourceMatches, filename)
	}
	matches = sourceMatches
	if len(matches) == 0 {
		panic(fmt.Sprintf("no Excel workbooks found in %q", config.Cfg.ExcelDir))
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
		panic(fmt.Sprintf("open Excel workbook %q failed: %v", excelFile, err))
	}
	defer f.Close()

	for _, sheetName := range f.GetSheetList() {
		if strings.HasPrefix(sheetName, "#") {
			// 跳过#开头的表格,备注类辅助表
			continue
		}

		if strings.HasSuffix(sheetName, "_Enum") {
			// 枚举表
			enumName := SplitEnumName(sheetName)
			ps := NewEnumParser(sheetName)
			ps.SetSourceFile(excelFile)
			if !ps.Parse(f) {
				panic(fmt.Sprintf("parse enum sheet %q in %q failed", sheetName, excelFile))
			}

			if !p.addEnumParser(enumName, ps) {
				panic(fmt.Sprintf("duplicate enum sheet %q in %q", sheetName, excelFile))
			}
		} else {
			// 数据表
			ps := NewSheetParser(sheetName)
			ps.SetSourceFile(excelFile)
			rows, err := f.GetRows(sheetName)
			if err != nil {
				panic(fmt.Sprintf("read data sheet %q in %q failed: %v", sheetName, excelFile, err))
			}
			ps.ParseRows(rows)
			ps.ParseHeadTags(f)

			if !p.addSheetParser(sheetName, ps) {
				panic(fmt.Sprintf("duplicate data sheet %q in %q", sheetName, excelFile))
			}
		}
	}
}

func (p *Parser) GetI18nFilePath() string {
	return filepath.Join(config.Cfg.ExcelDir, I18nSheetName+".xlsx")
}

func sameExistingFile(first, second string) bool {
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func (p *Parser) MergeI18n() {
	if !config.Cfg.EnableI18n {
		return
	}

	slog.Info("merge i18n start")
	// I18N.xlsx contains user-maintained translations. Validate all source data
	// before SaveAs can mutate that workbook.
	p.checks()
	i18nPath := p.GetI18nFilePath()

	//defer TimeCost("MergeI18n")()

	var i18n *I18nParser
	f, err := excelize.OpenFile(i18nPath)
	if err == nil {
		// 读取已存在的i18n文件
		ps := NewSheetParser(I18nSheetName)
		ps.SetSourceFile(i18nPath)
		rows, err := f.GetRows(I18nSheetName)
		if err != nil {
			panic(fmt.Sprintf("read I18N sheet %q in %q failed: %v", I18nSheetName, i18nPath, err))
		}
		ps.ParseRows(rows)

		i18n = NewI18nParser(ps)
		if err := f.Close(); err != nil {
			panic(fmt.Sprintf("close I18N workbook %q failed: %v", i18nPath, err))
		}
	} else if os.IsNotExist(err) {
		// 创建新的i18n文件
		i18n = NewI18nParser(nil)
		i18n.SetSourceFile(i18nPath)
	} else {
		panic(fmt.Sprintf("open I18N workbook %q failed: %v", i18nPath, err))
	}

	// 合并新的多语言数据
	for _, v := range p.sheets {
		i18n.MergeSheet(v)
	}
	i18n.checks(p)

	if err := i18n.WriteToExcel(i18nPath); err != nil {
		panic(fmt.Sprintf("write merged I18N workbook %q failed: %v", i18nPath, err))
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
	p.cleanupStaleOutputs()
}

func (p *Parser) hasSheetParser(sheetName string) bool {
	p.RLock()
	defer p.RUnlock()

	_, ok := p.sheets[sheetName]
	return ok
}

func (p *Parser) addSheetParser(sheetName string, sheet *SheetParser) bool {
	p.Lock()
	defer p.Unlock()
	if _, exists := p.sheets[sheetName]; exists {
		return false
	}
	p.sheets[sheetName] = sheet
	return true
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

func (p *Parser) addEnumParser(enumName string, enum *EnumParser) bool {
	p.Lock()
	defer p.Unlock()
	if _, exists := p.enums[enumName]; exists {
		return false
	}
	p.enums[enumName] = enum
	return true
}

func (p *Parser) getEnumParser(sheetName string) *EnumParser {
	p.RLock()
	defer p.RUnlock()

	return p.enums[sheetName]
}

func (p *Parser) checks() {
	if err := validateConfig(); err != nil {
		panic(fmt.Sprintf("invalid export configuration: %v", err))
	}
	for name := range p.sheets {
		if _, exists := p.enums[name]; exists {
			panic(fmt.Sprintf("data sheet and enum resolve to the same protobuf name %q", name))
		}
	}
	p.checkGeneratedNameCollisions()
	for _, sheet := range p.sheets {
		works.Go(func() {
			slog.Info("check Excel sheet", "sheet", sheet.sheetName, "source_file", sheet.sourceFile)
			sheet.checks(p)
		})
	}
	works.Wait()
	p.checkI18nKeyCollisions()
}

func (p *Parser) checkGeneratedNameCollisions() {
	protoNames := make(map[string]string, len(p.sheets)+len(p.enums))
	register := func(name string) {
		normalized := strings.ToLower(name)
		if previous, exists := protoNames[normalized]; exists && previous != name {
			panic(fmt.Sprintf("protobuf names %q and %q collide on case-insensitive filesystems", previous, name))
		}
		protoNames[normalized] = name
	}
	for name := range p.sheets {
		register(name)
	}
	for name := range p.enums {
		register(name)
	}

	for _, filter := range AllFilters {
		cfg := config.Cfg.Outs[FilterFullName[filter]]
		if cfg.CodeLanguage != "csharp" {
			continue
		}
		generated := make(map[string]string, len(protoNames))
		for _, name := range protoNames {
			filename := strings.ToLower(csharpProtoFilename(name))
			if previous, exists := generated[filename]; exists && previous != name {
				panic(fmt.Sprintf("protobuf names %q and %q both generate C# file %q", previous, name, csharpProtoFilename(name)))
			}
			generated[filename] = name
		}
	}
}

func (p *Parser) checkI18nKeyCollisions() {
	type location struct {
		sheet string
		field string
		row   int32
	}
	seen := make(map[string]location)
	for _, sheet := range p.sheets {
		for _, head := range sheet.headers {
			if !head.IsI18n() {
				continue
			}
			for rowIndex := range sheet.dataRows {
				value := sheet.getFiledValue(head.Name(), int32(rowIndex))
				if value == "" {
					continue
				}
				key := MakeI18nKey(sheet.sheetName, head.Name(), sheet.GetI18nPrimaryKey(int32(rowIndex)))
				if previous, exists := seen[key]; exists {
					panic(fmt.Sprintf("[%s field=%q] i18n key %q collides with sheet=%q field=%q excel_row=%d",
						sheet.configLocation(int32(rowIndex)), head.Name(), key,
						previous.sheet, previous.field, DataRowIndex2ExcelRow(previous.row)))
				}
				seen[key] = location{sheet: sheet.sheetName, field: head.Name(), row: int32(rowIndex)}
			}
		}
	}
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
			panic(fmt.Sprintf("create protobuf code output directory %q failed: %v", pbOutPath, err))
		}

		sss := p.protoFilesForFilter(filter)

		for _, filename := range sss {
			slog.Info("export pb file", "filename", filename)

			argInclude := fmt.Sprintf("--proto_path=%v", protoOutPath)
			argOut, supported := protobufOutArg(cfg.CodeLanguage, pbOutPath)
			if !supported {
				panic(fmt.Sprintf("unsupported protobuf code language %q for filter %q", cfg.CodeLanguage, filter))
			}

			works.Go(func() {
				args := []string{argInclude, argOut}
				if cfg.CodeLanguage == "golang" {
					args = append(args, "--go_opt=module="+cfg.GoModulePath)
				}
				args = append(args, filename)
				cmd := exec.Command("protoc", args...)
				output, err := cmd.CombinedOutput()
				if err != nil {
					panic(fmt.Sprintf("protoc failed for %q (%s): %v: %s", filename, cfg.CodeLanguage, err, strings.TrimSpace(string(output))))
				}
			})
		}
	}

	works.Wait()
}

func (p *Parser) protoFilesForFilter(filter string) []string {
	protoOutPath := config.Cfg.Outs[FilterFullName[filter]].ProtoPath
	files := make([]string, 0, len(p.sheets)+len(p.enums))
	seen := make(map[string]struct{}, len(p.sheets)+len(p.enums))
	for name := range p.sheets {
		seen[name] = struct{}{}
	}
	for name := range p.enums {
		seen[name] = struct{}{}
	}
	for name := range seen {
		files = append(files, filepath.Join(protoOutPath, name+".proto"))
	}
	sort.Strings(files)
	return files
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
			if !moduleCode.GenCode(p) {
				panic(fmt.Sprintf("generate golang loader code for filter %q failed", f))
			}
		case "csharp":
			moduleCode := NewCsharpLoaderCode(tplCodePath, codeOutPath)
			if !moduleCode.GenCode(p) {
				panic(fmt.Sprintf("generate csharp loader code for filter %q failed", f))
			}
		case "godot":
			moduleCode := NewGodotLoaderCode(tplCodePath, codeOutPath)
			if !moduleCode.GenCode(p) {
				panic(fmt.Sprintf("generate godot loader code for filter %q failed", f))
			}
		default:
			panic(fmt.Sprintf("unsupported load code language %q for filter %q", cfg.CodeLanguage, f))
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
