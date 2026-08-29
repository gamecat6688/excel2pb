package parser

import (
	"excel2pb/config"
	"fmt"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var reTrimTag = regexp.MustCompile(`[\n\t\r ]+`)
var protobufIdentifierPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
var generatedIdentifierPattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

var protobufReservedWords = map[string]struct{}{
	"syntax": {}, "import": {}, "weak": {}, "public": {}, "package": {}, "option": {},
	"optional": {}, "required": {}, "repeated": {}, "oneof": {}, "map": {}, "extensions": {},
	"to": {}, "max": {}, "reserved": {}, "service": {}, "rpc": {}, "stream": {}, "returns": {},
	"message": {}, "enum": {}, "extend": {}, "group": {}, "true": {}, "false": {},
}

var primitiveProtoTypes = map[string]struct{}{
	"int32": {}, "int64": {}, "float": {}, "double": {}, "bool": {}, "string": {},
	TimestampName: {}, I18nName: {},
}

func isValidProtobufIdentifier(value string) bool {
	if !protobufIdentifierPattern.MatchString(value) {
		return false
	}
	_, reserved := protobufReservedWords[value]
	return !reserved
}

func isStableGeneratedIdentifier(value string) bool {
	return generatedIdentifierPattern.MatchString(value)
}

/*
 * 数据表处理
 */

// 客户端或服务器
type Sheet struct {
	sheetName  string
	sourceFile string

	filter string // c or s

	// 表头
	headers        []Head
	headersIndexes map[string]int32 // key is name, value is index

	// 数据
	headRows [][]string
	dataRows [][]string
}

func (s *Sheet) IsClient() bool {
	return s.filter == ClientFlag
}

func (s *Sheet) IsServer() bool {
	return s.filter == ServerFlag
}

func (s *Sheet) HasData() bool {
	return len(s.dataRows) > 0
}

// //////////////////////
// SheetParser
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

func (s *SheetParser) SetSourceFile(sourceFile string) {
	s.sourceFile = sourceFile
	s.logger = slog.Default().With("sheet", s.sheetName, "source_file", sourceFile)
}

func (s *SheetParser) configLocation(rowIndex int32) string {
	location := fmt.Sprintf("file=%q sheet=%q", s.sourceFile, s.sheetName)
	if rowIndex >= 0 {
		location += fmt.Sprintf(" excel_row=%d", DataRowIndex2ExcelRow(rowIndex))
	}
	return location
}

// getFieldColIndex 获得指定字段名的列索引
func (s *SheetParser) getFieldColIndex(name string) int32 {
	colIndex, ok := s.headersIndexes[name]
	if !ok {
		panic(fmt.Sprintf("[%s] field %q not found", s.configLocation(-1), name))
	}
	return colIndex
}

// GetFiled 获得指定字段名的列索引
func (s *SheetParser) GetFiled(colIndex int32) Head {
	if colIndex < 0 || int(colIndex) >= len(s.headers) {
		panic(fmt.Sprintf("[%s] field index %v is outside header range 0..%v", s.configLocation(-1), colIndex, len(s.headers)-1))
	}
	return s.headers[colIndex]
}

func (s *SheetParser) GetPrimaryKeys() (rv []Head) {
	for _, v := range s.headers {
		if v.IsPrimaryKey() {
			rv = append(rv, v)
		}
	}
	return
}

func (s *SheetParser) GetI18nPrimaryKey(rowIdx int32) string {
	var pkValue string
	pks := s.GetPrimaryKeys()
	for _, v := range pks {
		pkValue += "_" + s.getFiledValue(v.Name(), rowIdx)
	}
	return strings.TrimLeft(pkValue, "_")
}

func (s *SheetParser) ParseRows(rows [][]string) {
	s.parseHeader(rows)
	s.parseData(rows)
}

func (s *SheetParser) ParseHeadTags(f *excelize.File) {
	comments, err := f.GetComments(s.sheetName)
	if err != nil {
		s.logger.Warn("read sheet comments failed", "error", err)
		return
	}
	for _, v := range comments {
		col, row, err := excelize.CellNameToCoordinates(v.Cell)
		if err != nil {
			s.logger.Warn("skip comment with invalid cell reference", "cell", v.Cell, "error", err)
			continue
		}
		if row != 1 {
			// 出需要处理第一行的批注
			continue
		}

		// 处理批注
		if col < 1 || col > len(s.headers) {
			s.logger.Warn("skip comment outside sheet header", "cell", v.Cell)
			continue
		}

		// Comments written by other Excel libraries may have no rich-text runs.
		txtTag := v.Text
		if txtTag == "" {
			var builder strings.Builder
			for _, paragraph := range v.Paragraph {
				builder.WriteString(paragraph.Text)
			}
			txtTag = builder.String()
		}
		txtTag = reTrimTag.ReplaceAllString(txtTag, "")
		if txtTag == "" {
			continue
		}

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

func (s *SheetParser) isHeader(rowIndex int32) bool {
	return rowIndex < HeadCount
}

func (s *SheetParser) clearHeaders() {
	s.headers = nil
	s.headersIndexes = map[string]int32{}
	s.headRows = nil
}

func (s *SheetParser) parseHeader(rows [][]string) {
	s.clearHeaders()

	if len(rows) < HeadCount || len(rows[0]) == 0 {
		panic(fmt.Sprintf("[%s] header is empty or has fewer than %d rows", s.configLocation(-1), HeadCount))
	}

	// 预先分配字段数量
	s.headers = make([]Head, len(rows[0]))

	for i, row := range rows {
		if !s.isHeader(int32(i)) {
			break
		}
		if len(row) > len(s.headers) {
			panic(fmt.Sprintf("[%s] header row %d has %d columns, but the field-name row has %d", s.configLocation(-1), i+1, len(row), len(s.headers)))
		}

		for j, cell := range row {
			s.headers[j].info[i] = cell
		}

		s.headRows = append(s.headRows, row)
	}

	// build indexes
	for colIndex, head := range s.headers {
		s.headersIndexes[head.Name()] = int32(colIndex)
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
		if len(row) > len(s.headers) {
			panic(fmt.Sprintf("[%s] data row has %d columns, but the header has %d",
				s.configLocation(int32(i-HeadCount)), len(row), len(s.headers)))
		}

		s.dataRows = append(s.dataRows, row)
	}
}

// TODO 优化这个方法
// 可以缓存成map，减少重复计算
func (s *SheetParser) hasExistFiledValue(root *Parser, field Head, fieldValue string) bool {
	canonical := s.canonicalFieldValue(root, field, 0, fieldValue)
	for rowIndex := range s.dataRows {
		value := s.getFiledValue(field.Name(), int32(rowIndex))
		if s.canonicalFieldValue(root, field, int32(rowIndex), value) == canonical {
			return true
		}
	}
	return false
}

func (s *SheetParser) getFiledValue(filedName string, rowIndex int32) string {
	colIndex := s.getFieldColIndex(filedName)
	if int(colIndex) >= len(s.dataRows[rowIndex]) {
		return ""
	}
	return s.dataRows[rowIndex][colIndex]
}

func (s *SheetParser) checks(root *Parser) {
	s.checkHeaderSchema(root)
	s.checkPrimaryKey(root)
	s.checkUnique(root)
	s.checkCustomMessageValues(root)
	s.checkValueTypes(root)
	s.checkTags(root)
}

func (s *SheetParser) checkHeaderSchema(root *Parser) {
	if !isValidProtobufIdentifier(s.sheetName) {
		panic(fmt.Sprintf("[%s] invalid protobuf sheet name %q", s.configLocation(-1), s.sheetName))
	}
	if !isStableGeneratedIdentifier(s.sheetName) {
		panic(fmt.Sprintf("[%s] sheet name %q must be PascalCase without underscores for generated loaders", s.configLocation(-1), s.sheetName))
	}
	hasI18n := false
	fieldNames := make(map[string]struct{}, len(s.headers))
	for _, head := range s.headers {
		if !isValidProtobufIdentifier(head.Name()) {
			panic(fmt.Sprintf("[%s] invalid protobuf field name %q", s.configLocation(-1), head.Name()))
		}
		if !isStableGeneratedIdentifier(head.Name()) {
			panic(fmt.Sprintf("[%s] field name %q must be PascalCase without underscores for generated loaders", s.configLocation(-1), head.Name()))
		}
		if _, exists := fieldNames[head.Name()]; exists {
			panic(fmt.Sprintf("[%s] duplicate field name %q", s.configLocation(-1), head.Name()))
		}
		fieldNames[head.Name()] = struct{}{}

		parts := strings.Fields(head.Type())
		if len(parts) == 0 || len(parts) > 2 {
			panic(fmt.Sprintf("[%s field=%q] invalid field type %q", s.configLocation(-1), head.Name(), head.Type()))
		}
		if len(parts) == 2 {
			switch parts[0] {
			case PrimaryKeyName, UniqueName, RepeatedName:
			default:
				panic(fmt.Sprintf("[%s field=%q] invalid field modifier %q", s.configLocation(-1), head.Name(), parts[0]))
			}
		}
		baseType := head.BaseType()
		if _, primitive := primitiveProtoTypes[baseType]; !primitive && !root.hasSheetParser(baseType) && !root.hasEnumParser(baseType) {
			panic(fmt.Sprintf("[%s field=%q] unsupported type %q", s.configLocation(-1), head.Name(), baseType))
		}
		if head.IsRepeated() && head.IsI18n() {
			panic(fmt.Sprintf("[%s field=%q] repeated i18n fields are not supported", s.configLocation(-1), head.Name()))
		}
		if (head.IsPrimaryKey() || head.IsUnique()) && root.hasSheetParser(baseType) {
			panic(fmt.Sprintf("[%s field=%q] message type %q cannot be used as a primary or unique key", s.configLocation(-1), head.Name(), baseType))
		}
		switch head.ExportFilter() {
		case ClientFlag, ServerFlag, ClientFlag + ServerFlag:
		default:
			panic(fmt.Sprintf("[%s field=%q] invalid export filter %q: expected c, s, or cs",
				s.configLocation(-1), head.Name(), head.ExportFilter()))
		}
		if head.IsI18n() {
			hasI18n = true
			if head.IsPrimaryKey() {
				panic(fmt.Sprintf("[%s field=%q] i18n field cannot be a primary key", s.configLocation(-1), head.Name()))
			}
		}
	}
	if hasI18n && len(s.GetPrimaryKeys()) == 0 {
		panic(fmt.Sprintf("[%s] sheet with i18n fields must define a primary key", s.configLocation(-1)))
	}
	if s.HasData() {
		pks := s.GetPrimaryKeys()
		if len(pks) == 0 {
			panic(fmt.Sprintf("[%s] data sheet must define a primary key", s.configLocation(-1)))
		}
		for _, pk := range pks {
			for _, filter := range AllFilters {
				if !pk.IsFilter(filter) {
					panic(fmt.Sprintf("[%s field=%q] primary key must be exported to every configured target", s.configLocation(-1), pk.Name()))
				}
			}
		}
	}
}

func (s *SheetParser) checkCustomMessageValues(root *Parser) {
	for _, head := range s.headers {
		if !head.IsCustomMessage(root) {
			continue
		}

		customParser := root.getSheetParser(head.BaseType())
		for rowIndex := range s.dataRows {
			value := s.getFiledValue(head.Name(), int32(rowIndex))
			for _, customRow := range SplitCustomValue(value) {
				s.checkCustomFieldCount(head, int32(rowIndex), value, customRow, customParser)
				for colIndex, subValue := range customRow {
					subHead := customParser.GetFiled(int32(colIndex))
					if subHead.IsCustomMessage(root) || subHead.IsRepeated() || subHead.IsI18n() {
						panic(fmt.Sprintf("[%s field=%q] nested field %s.%s uses unsupported type %q",
							s.configLocation(int32(rowIndex)), head.Name(), customParser.sheetName, subHead.Name(), subHead.Type()))
					}
					if subValue == "" && subHead.BaseType() != "string" {
						panic(fmt.Sprintf("[%s field=%q] nested scalar %s.%s is empty",
							s.configLocation(int32(rowIndex)), head.Name(), customParser.sheetName, subHead.Name()))
					}
					customParser.TypeNameToValue(root, subHead, int32(rowIndex), subValue)
				}
			}
		}
	}
}

func (s *SheetParser) checkCustomFieldCount(fd Head, rowIdx int32, value string, customRow []string, customParser *SheetParser) {
	expected := len(customParser.headers)
	if len(customRow) == expected {
		return
	}

	panic(fmt.Sprintf("[%s field=%q] invalid composite value: %v expects %v fields, got %v in %q",
		s.configLocation(rowIdx), fd.Name(), fd.BaseType(), expected, len(customRow), value))
}

// checkPrimaryKey 主键唯一性检测
// 单主键：该列唯一；多主键：按组合唯一（符合复合主键语义，单列可重复，组合唯一）
func (s *SheetParser) checkPrimaryKey(root *Parser) {
	pks := s.GetPrimaryKeys()
	if len(pks) == 0 {
		return
	}

	seen := make(map[string]bool)
	for rowIndex := range s.dataRows {
		parts := make([]string, 0, len(pks))
		for _, pk := range pks {
			value := s.getFiledValue(pk.Name(), int32(rowIndex))
			if strings.TrimSpace(value) == "" {
				panic(fmt.Sprintf("[%s field=%q] primary key value is empty",
					s.configLocation(int32(rowIndex)), pk.Name()))
			}
			parts = append(parts, s.canonicalFieldValue(root, pk, int32(rowIndex), value))
		}
		// \x00 作为分隔符，避免不同组合拼接后碰撞
		key := strings.Join(parts, "\x00")
		if seen[key] {
			names := make([]string, 0, len(pks))
			for _, pk := range pks {
				names = append(names, pk.Name())
			}
			panic(fmt.Sprintf("[%s] duplicate primary key (%v)=(%v)",
				s.configLocation(int32(rowIndex)), strings.Join(names, ","), strings.Join(parts, ",")))
		}
		seen[key] = true
	}
}

// checkUnique 当前表的数据唯一性检测
func (s *SheetParser) checkUnique(root *Parser) {
	for _, head := range s.headers {
		if head.IsUnique() {
			values := make(map[string]struct{})
			for rowIndex := range s.dataRows {
				value := s.getFiledValue(head.Name(), int32(rowIndex))
				canonical := s.canonicalFieldValue(root, head, int32(rowIndex), value)
				if _, exists := values[canonical]; exists {
					panic(fmt.Sprintf("[%s field=%q] duplicate unique value %q",
						s.configLocation(int32(rowIndex)), head.Name(), value))
				}
				values[canonical] = struct{}{}
			}
		}
	}
}

func (s *SheetParser) canonicalFieldValue(root *Parser, head Head, rowIndex int32, value string) string {
	parsed := s.TypeNameToValue(root, head, rowIndex, value)
	return fmt.Sprintf("%v", parsed.Interface())
}

func (s *SheetParser) checkValueTypes(root *Parser) {
	for _, head := range s.headers {
		if head.IsCustomMessage(root) {
			continue
		}
		for rowIndex := range s.dataRows {
			value := s.getFiledValue(head.Name(), int32(rowIndex))
			if value == "" {
				if head.IsRepeated() || head.BaseType() == "string" || head.IsI18n() {
					continue
				}
				panic(fmt.Sprintf("[%s field=%q] scalar value is empty", s.configLocation(int32(rowIndex)), head.Name()))
			}
			for _, item := range SplitBaseValue(value) {
				s.TypeNameToValue(root, head, int32(rowIndex), item)
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
				embedSheetName, embedFiledName, fkSheetName, fkFiledName, err := tag.ParseForeignKey()
				if err != nil {
					panic(fmt.Sprintf("[%s field=%q] %v", s.configLocation(-1), head.Name(), err))
				}

				if len(embedSheetName) == 0 {
					// 没有内嵌结构

					// 检测外键的表是否存在
					refSheetParser := root.getSheetParser(fkSheetName)
					if refSheetParser == nil {
						panic(fmt.Sprintf("[%s field=%q] foreign key target sheet %q not found", s.configLocation(-1), head.Name(), fkSheetName))
					}
					fkIndex, exists := refSheetParser.headersIndexes[fkFiledName]
					if !exists {
						panic(fmt.Sprintf("[%s field=%q] foreign key target field %q.%q not found", s.configLocation(-1), head.Name(), fkSheetName, fkFiledName))
					}
					fkHead := refSheetParser.GetFiled(fkIndex)
					checkForeignKeyTypes(s, head, refSheetParser, fkHead)

					// 检测外键的字段值是否存在（支持repeated）
					for rowIndex, _ := range s.dataRows {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						if thisValue == "" && !head.IsRepeated() {
							panic(fmt.Sprintf("[%s field=%q] foreign key value is empty", s.configLocation(int32(rowIndex)), head.Name()))
						}
						values := SplitBaseValue(thisValue)
						for _, checkValue := range values {
							if !refSheetParser.hasExistFiledValue(root, fkHead, checkValue) {
								panic(fmt.Sprintf("[%s field=%q] foreign key value %q not found in %s.%s",
									s.configLocation(int32(rowIndex)), head.Name(), checkValue, fkSheetName, fkFiledName))
							}
						}
					}

				} else {
					// 有内嵌结构

					// 检测外键的表是否存在
					refSheetParser := root.getSheetParser(fkSheetName)
					customParser := root.getSheetParser(embedSheetName)
					if refSheetParser == nil || customParser == nil {
						panic(fmt.Sprintf("[%s field=%q] embedded foreign key sheet not found: embedded=%q target=%q",
							s.configLocation(-1), head.Name(), embedSheetName, fkSheetName))
					}
					embedIndex, exists := customParser.headersIndexes[embedFiledName]
					if !exists {
						panic(fmt.Sprintf("[%s field=%q] embedded foreign key field %q.%q not found", s.configLocation(-1), head.Name(), embedSheetName, embedFiledName))
					}
					fkIndex, exists := refSheetParser.headersIndexes[fkFiledName]
					if !exists {
						panic(fmt.Sprintf("[%s field=%q] foreign key target field %q.%q not found", s.configLocation(-1), head.Name(), fkSheetName, fkFiledName))
					}
					embedHead := customParser.GetFiled(embedIndex)
					fkHead := refSheetParser.GetFiled(fkIndex)
					checkForeignKeyTypes(customParser, embedHead, refSheetParser, fkHead)

					// 检测外键的字段值是否存在（支持repeated）
					for rowIndex, _ := range s.dataRows {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						values := SplitCustomValue(thisValue)
						for _, checkMsg := range values {
							idx := customParser.getFieldColIndex(embedFiledName)
							s.checkCustomFieldCount(head, int32(rowIndex), thisValue, checkMsg, customParser)
							checkValue := checkMsg[idx]
							if !refSheetParser.hasExistFiledValue(root, fkHead, checkValue) {
								panic(fmt.Sprintf("[%s field=%q] embedded foreign key value %q not found in %s.%s",
									s.configLocation(int32(rowIndex)), head.Name(), checkValue, fkSheetName, fkFiledName))
							}
						}
					}
				}

			case TagIndexName:
				// TODO
			default:
				panic(fmt.Sprintf("[%s field=%q] unsupported tag %q", s.configLocation(-1), head.Name(), tag))
			}
		}
	}
}

func checkForeignKeyTypes(sourceSheet *SheetParser, source Head, targetSheet *SheetParser, target Head) {
	if target.IsRepeated() {
		panic(fmt.Sprintf("[%s field=%q] repeated fields cannot be foreign key targets",
			targetSheet.configLocation(-1), target.Name()))
	}
	uniqueTarget := target.IsUnique() || (target.IsPrimaryKey() && len(targetSheet.GetPrimaryKeys()) == 1)
	if !uniqueTarget {
		panic(fmt.Sprintf("[%s field=%q] foreign key target must be unique or the sole primary key",
			targetSheet.configLocation(-1), target.Name()))
	}
	if source.BaseType() != target.BaseType() {
		panic(fmt.Sprintf("[%s field=%q] foreign key type %q does not match [%s field=%q] type %q",
			sourceSheet.configLocation(-1), source.Name(), source.BaseType(),
			targetSheet.configLocation(-1), target.Name(), target.BaseType()))
	}
	if source.IsI18n() || target.IsI18n() {
		panic(fmt.Sprintf("[%s field=%q] i18n fields cannot be used as foreign keys", sourceSheet.configLocation(-1), source.Name()))
	}
}

func (s *SheetParser) SplitByFilter(filterName string) *SheetParser {
	ns := &SheetParser{
		logger: s.logger,
		Sheet: Sheet{
			headersIndexes: make(map[string]int32),
		},
	}
	ns.sheetName = s.sheetName
	ns.sourceFile = s.sourceFile
	ns.filter = filterName

	// 复制过滤后的表头
	for i, head := range s.headers {
		if head.IsFilter(filterName) {
			ns.headers = append(ns.headers, head)
			ns.headersIndexes[head.Name()] = int32(i)
		}
	}

	// 复制过滤后的数据
	for _, row := range s.dataRows {
		var newRow []string
		for _, head := range ns.headers {
			colIndex := s.getFieldColIndex(head.Name())
			if int(colIndex) >= len(row) {
				continue
			}
			newRow = append(newRow, row[colIndex])
		}
		ns.dataRows = append(ns.dataRows, newRow)
	}

	// 重建表头索引
	for i, head := range ns.headers {
		ns.headersIndexes[head.Name()] = int32(i)
	}

	return ns
}

func (s *SheetParser) getPackageName() string {
	return config.Cfg.Outs[FilterFullName[s.filter]].PackageName
}

func goPackagePathForFilter(filter string) string {
	cfg := config.Cfg.Outs[FilterFullName[filter]]
	if cfg.GoPackagePath != "" {
		return cfg.GoPackagePath
	}
	return cfg.PackageName
}

func (s *SheetParser) getGoPackagePath() string {
	return goPackagePathForFilter(s.filter)
}

func (s *SheetParser) getProtoOutPath() string {
	return config.Cfg.Outs[FilterFullName[s.filter]].ProtoPath
}

func (s *SheetParser) getDataOutPath() string {
	return config.Cfg.Outs[FilterFullName[s.filter]].DataPath
}

func (s *SheetParser) getDataExtension() string {
	return config.Cfg.Outs[FilterFullName[s.filter]].DataExt
}

func (s *SheetParser) getPbOutPath() string {
	return config.Cfg.Outs[FilterFullName[s.filter]].PbPath
}

func (s *SheetParser) getCodeLanguage() string {
	return config.Cfg.Outs[FilterFullName[s.filter]].CodeLanguage
}

func (s *SheetParser) getDataFilePath() string {
	return filepath.Join(s.getDataOutPath(), s.sheetName+s.getDataExtension())
}

func (s *SheetParser) getProtoFilePath() string {
	return filepath.Join(s.getProtoOutPath(), s.getProtoName())
}

func (s *SheetParser) getProtoName() string {
	return s.sheetName + ".proto"
}

func (s *SheetParser) getImportMessages(root *Parser) (rv []string) {
	exists := map[string]bool{}
	for _, v := range s.headers {
		baseType := v.BaseType()
		if root.hasSheetParser(baseType) || root.hasEnumParser(baseType) {
			_, ok := exists[baseType]
			if !ok {
				rv = append(rv, baseType)
				exists[baseType] = true
			}
		}
	}

	return
}

func (s *SheetParser) getAllProtoFiles(root *Parser) []string {
	// 获得当前表格所有的proto文件，包含依赖文件
	var protoNames []string
	protoNames = append(protoNames, s.getProtoName())
	importMessages := s.getImportMessages(root)
	for _, v := range importMessages {
		protoNames = append(protoNames, v+".proto")
	}
	return protoNames
}

// import "playerstate/ExampleNonBasicPart.proto";
func (s *SheetParser) ExportProto(root *Parser) {
	// 解析模板
	tmpl, err := template.New("proto").Parse(ProtoMessageTemplate)
	if err != nil {
		panic(fmt.Sprintf("[%s] parse proto message template failed: %v", s.configLocation(-1), err))
	}

	m := &ProtoMessageModel{
		PackageName:   s.getPackageName(),
		GoPackagePath: s.getGoPackagePath(),
		MessageName:   s.sheetName,
	}

	// 数据驱动模板
	importMessages := s.getImportMessages(root)
	for _, v := range importMessages {
		// 引用其他proto
		m.Imports = append(m.Imports, ImportModel{
			ProtoPath: fmt.Sprintf(`"%v%v.proto";`, config.Cfg.ProtoImportPath, v),
		})
	}

	for t, v := range s.headers {
		m.Fields = append(m.Fields, FieldModel{
			ProtoType: v.ProtoType(),
			FieldName: v.Name(),
			FieldTag:  int32(t) + 1,
			Comment:   v.Desc(),
		})
	}

	// 创建proto文件
	outPath := s.getProtoOutPath()
	if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
		panic(fmt.Sprintf("[%s] create proto output directory %q failed: %v", s.configLocation(-1), outPath, err))
	}

	f, err := os.Create(s.getProtoFilePath())
	if err != nil {
		panic(fmt.Sprintf("[%s] create proto file %q failed: %v", s.configLocation(-1), s.getProtoFilePath(), err))
	}

	// 执行模板,输出文件
	err = tmpl.Execute(f, m)
	if err != nil {
		_ = f.Close()
		panic(fmt.Sprintf("[%s] render proto file %q failed: %v", s.configLocation(-1), s.getProtoFilePath(), err))
	}
	if err := f.Close(); err != nil {
		panic(fmt.Sprintf("[%s] close proto file %q failed: %v", s.configLocation(-1), s.getProtoFilePath(), err))
	}
}

// 例子： protoc --proto_path=".\assets\out_proto\server" --go_out=".\assets\out_pb\server" "assets\out_proto\server\Item.proto"
//func (s *SheetParser) ExportPb(root *Parser) {
//	// make dirs
//	outPath := s.getPbOutPath()
//	os.MkdirAll(outPath, os.ModePerm)
//
//	argInclude := fmt.Sprintf("--proto_path=%v", s.getProtoOutPath())
//	argOut := ""
//
//	switch s.getCodeLanguage() {
//	case "golang":
//		argOut = fmt.Sprintf("--go_out=%v", s.getPbOutPath())
//	case "csharp":
//		argOut = fmt.Sprintf("--csharp_out=%v", s.getPbOutPath())
//	case "java":
//		argOut = fmt.Sprintf("--java_out=%v", s.getPbOutPath())
//	case "cpp":
//		argOut = fmt.Sprintf("--cpp_out=%v", s.getPbOutPath())
//	}
//
//	cmd := exec.Command("protoc", argInclude, argOut, s.getProtoFilePath())
//	err := cmd.Run()
//	if err != nil {
//		log.Fatalf("protoc failed: %v", err)
//	}
//}

func (s *SheetParser) ExportData(root *Parser) {
	if !s.HasData() {
		return
	}

	// 获得当前表格所有的proto文件，包含依赖文件
	protoNames := s.getAllProtoFiles(root)
	resolver, err := parseProtoFile(protoNames, []string{s.getProtoOutPath()})
	if err != nil {
		panic(fmt.Sprintf("[%s] compile generated proto files %v failed: %v", s.configLocation(-1), protoNames, err))
	}

	// 2. 获取消息描述符
	configMsgName := fmt.Sprintf("%v.%v", s.getPackageName(), s.sheetName)
	configMsgType, err := getMessageType(resolver, configMsgName+"Config")
	if err != nil {
		panic(fmt.Sprintf("[%s] resolve protobuf message %q failed: %v", s.configLocation(-1), configMsgName+"Config", err))
	}
	recordMsgType, err := getMessageType(resolver, configMsgName)
	if err != nil {
		panic(fmt.Sprintf("[%s] resolve protobuf message %q failed: %v", s.configLocation(-1), configMsgName, err))
	}

	// 5. 构造多个 record 数据
	var records []protoreflect.Message
	for rowIdx, row := range s.dataRows {
		record := recordMsgType.New()

		for colIdx, value := range row {
			fd := s.GetFiled(int32(colIdx))
			pbField := record.Descriptor().Fields().ByName(protoreflect.Name(fd.Name()))
			if fd.IsCustomMessage(root) {
				// message字段
				subMsgName := fmt.Sprintf("%v.%v", s.getPackageName(), fd.BaseType())
				subMsgType, err2 := getMessageType(resolver, subMsgName)
				if err2 != nil {
					panic(fmt.Sprintf("[%s] resolve nested protobuf message %q failed: %v", s.configLocation(int32(rowIdx)), subMsgName, err2))
				}

				customs := s.procCustomProtoType(root, subMsgType, fd, int32(rowIdx), value)
				if fd.IsRepeated() {
					fdList := record.Mutable(pbField).List()
					for _, val := range customs {
						fdList.Append(val)
					}
				} else if customs != nil {
					record.Set(pbField, customs[0])
				}
			} else {
				// 基础类型和enum字段
				arrValues := s.procBaseProtoType(root, fd, int32(rowIdx), value)
				if fd.IsRepeated() {
					fdList := record.Mutable(pbField).List()
					for _, val := range arrValues {
						fdList.Append(val)
					}
				} else if arrValues != nil {
					record.Set(pbField, arrValues[0])
				}
			}
		}

		records = append(records, record)
	}

	// 3. 创建动态消息并填充数据
	configMsg := configMsgType.New()
	pbConfigField := configMsg.Descriptor().Fields().ByName("Records")
	recordList := configMsg.Mutable(pbConfigField).List()
	for _, r := range records {
		recordList.Append(protoreflect.ValueOfMessage(r))
	}

	// 4. 序列化
	data, err := proto.Marshal(configMsg.(proto.Message))
	if err != nil {
		panic(fmt.Sprintf("[%s] marshal protobuf data failed: %v", s.configLocation(-1), err))
	}

	// make dirs
	outPath := s.getDataOutPath()
	if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
		panic(fmt.Sprintf("[%s] create data output directory %q failed: %v", s.configLocation(-1), outPath, err))
	}

	// write dataRows file
	dataFilePath := s.getDataFilePath()
	if err := os.WriteFile(dataFilePath, data, os.ModePerm); err != nil {
		panic(fmt.Sprintf("[%s] write protobuf data %q failed: %v", s.configLocation(-1), dataFilePath, err))
	}
}

func (s *SheetParser) ExportCode(root *Parser) {
	if !s.HasData() {
		return
	}

	cfg := config.Cfg.Outs[FilterFullName[s.filter]]
	tplCodePath := config.Cfg.TplCodePaths[cfg.CodeLanguage]
	codeOutPath := config.Cfg.CodeOutPaths[cfg.CodeLanguage]
	switch cfg.CodeLanguage {
	case "golang":
		moduleCode := NewGolangModuleCode(tplCodePath, codeOutPath)
		if !moduleCode.GenCode(root, s) {
			panic(fmt.Sprintf("[%s] generate golang module code failed", s.configLocation(-1)))
		}
	case "csharp":
		moduleCode := NewCsharpModuleCode(tplCodePath, codeOutPath)
		if !moduleCode.GenCode(root, s) {
			panic(fmt.Sprintf("[%s] generate csharp module code failed", s.configLocation(-1)))
		}
	case "godot":
		moduleCode := NewGodotModuleCode(tplCodePath, codeOutPath)
		if !moduleCode.GenCode(root, s) {
			panic(fmt.Sprintf("[%s] generate godot module code failed", s.configLocation(-1)))
		}
	default:
		panic(fmt.Sprintf("[%s] unsupported module code language %q", s.configLocation(-1), cfg.CodeLanguage))
	}
}

func (s *SheetParser) procBaseProtoType(root *Parser, fd Head, rowIdx int32, value string) []protoreflect.Value {
	if value == "" && !fd.IsRepeated() && fd.BaseType() != "string" && !fd.IsI18n() {
		panic(fmt.Sprintf("[%s field=%q] scalar value is empty", s.configLocation(rowIdx), fd.Name()))
	}
	var arrValue []protoreflect.Value
	for _, v := range SplitBaseValue(value) {
		arrValue = append(arrValue, s.TypeNameToValue(root, fd, rowIdx, v))
	}
	return arrValue
}

func (s *SheetParser) procCustomProtoType(root *Parser, msgType protoreflect.MessageType, fd Head, rowIdx int32, value string) []protoreflect.Value {
	var customs []protoreflect.Value

	customParser := root.getSheetParser(fd.BaseType())

	// 构造多个子结构
	customRows := SplitCustomValue(value)
	for _, customRow := range customRows {
		s.checkCustomFieldCount(fd, rowIdx, value, customRow, customParser)
		customMsg := msgType.New()
		for idx, subValue := range customRow {
			subFd := customParser.GetFiled(int32(idx))
			if s.filter != "" && !subFd.IsFilter(s.filter) {
				continue
			}
			pbField := customMsg.Descriptor().Fields().ByName(protoreflect.Name(subFd.Name()))
			if pbField == nil {
				panic(fmt.Sprintf("[%s field=%q] nested protobuf field %q is missing from filter %q",
					s.configLocation(rowIdx), fd.Name(), subFd.Name(), s.filter))
			}
			customMsg.Set(pbField, s.TypeNameToValue(root, subFd, rowIdx, subValue))
		}
		customs = append(customs, protoreflect.ValueOfMessage(customMsg))
	}

	return customs
}

func (s *SheetParser) TypeNameToValue(root *Parser, fd Head, rowIdx int32, value string) protoreflect.Value {
	sheetName := s.sheetName
	headName := fd.Name()
	typeName := fd.BaseType()

	switch typeName {
	case "string":
		return protoreflect.ValueOfString(value)
	case "int32":
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse int32 failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
		}
		return protoreflect.ValueOfInt32(int32(parsed))
	case "int64":
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse int64 failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
		}
		return protoreflect.ValueOfInt64(parsed)
	case "float":
		parsed, err := strconv.ParseFloat(value, 32)
		if err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse float failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
		}
		return protoreflect.ValueOfFloat32(float32(parsed))
	case "double":
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse double failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
		}
		return protoreflect.ValueOfFloat64(parsed)
	case "bool":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse bool failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
		}
		return protoreflect.ValueOfBool(parsed)
	case TimestampName:
		if _, err := time.Parse("2006-01-02 15:04:05", value); err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse timestamp failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
		}
		timeOfZone := DataTimeToRFC3339(value, config.Cfg.TimeZone)
		t, err := time.Parse(time.RFC3339, timeOfZone)
		if err != nil {
			panic(fmt.Sprintf("[%s field=%q] parse timestamp failed: value=%q RFC3339=%q error=%v", s.configLocation(rowIdx), headName, value, timeOfZone, err))
		}
		return protoreflect.ValueOfInt64(t.Unix())
	case I18nName:
		pkValue := s.GetI18nPrimaryKey(rowIdx)
		i18nKey := MakeI18nKey(sheetName, headName, pkValue)
		return protoreflect.ValueOfString(i18nKey)
	default:
		if root.hasEnumParser(typeName) {
			// 枚举类型处理
			if IsNumber(value) {
				// 配置的枚举值
				parsed, err := strconv.ParseInt(value, 10, 32)
				if err != nil {
					panic(fmt.Sprintf("[%s field=%q] parse enum number failed: value=%q error=%v", s.configLocation(rowIdx), headName, value, err))
				}
				enumParser := root.getEnumParser(typeName)
				if !enumParser.hasEnumValue(int32(parsed)) {
					panic(fmt.Sprintf("[%s field=%q] enum %q does not define numeric value %d", s.configLocation(rowIdx), headName, typeName, parsed))
				}
				return protoreflect.ValueOfEnum(protoreflect.EnumNumber(parsed))
			} else {
				// 配置的枚举名
				enumParser := root.getEnumParser(typeName)
				enumValue := enumParser.getEnumValue(value)
				return protoreflect.ValueOfEnum(protoreflect.EnumNumber(enumValue))
			}
		} else {
			panic(fmt.Sprintf("[%s field=%q] unsupported type %q", s.configLocation(rowIdx), headName, typeName))
		}
	}
}
