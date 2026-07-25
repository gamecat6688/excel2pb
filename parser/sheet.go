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
	"strings"
	"text/template"
	"time"
)

var reTrimTag = regexp.MustCompile(`[\n\t\r ]+`)

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

		s.dataRows = append(s.dataRows, row)
	}
}

// 返回
func (s *SheetParser) checkFiledValueIsUnique(filedName string) (value string, ok bool) {
	// 检查指定列的数据，是否存在重复值
	_ = s.getFieldColIndex(filedName)
	values := make(map[string]bool)
	for rowIndex := range s.dataRows {
		v := s.getFiledValue(filedName, int32(rowIndex))
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
	_ = s.getFieldColIndex(filedName)
	for rowIndex := range s.dataRows {
		value := s.getFiledValue(filedName, int32(rowIndex))
		if value == filedValue {
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
	s.checkPrimaryKey(root)
	s.checkUnique(root)
	s.checkCustomMessageValues(root)
	s.checkTags(root)
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
			parts = append(parts, s.getFiledValue(pk.Name(), int32(rowIndex)))
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
			value, ok := s.checkFiledValueIsUnique(head.Name())
			if !ok {
				panic(fmt.Sprintf("[%s field=%q] duplicate unique value %q",
					s.configLocation(-1), head.Name(), value))
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
				_ = embedFiledName

				if len(embedSheetName) == 0 {
					// 没有内嵌结构

					// 检测外键的表是否存在
					refSheetParser := root.getSheetParser(fkSheetName)
					if refSheetParser == nil {
						s.logger.Error("foreign key target sheet not found", "field", head.Name(), "target_sheet", fkSheetName, "target_field", fkFiledName)
						continue
					}

					// 检测外键的字段值是否存在（支持repeated）
					for rowIndex, _ := range s.dataRows {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						values := SplitBaseValue(thisValue)
						for _, checkValue := range values {
							if !refSheetParser.hasExistFiledValue(fkFiledName, checkValue) {
								s.logger.Error("foreign key value not found", "field", head.Name(), "excel_row", DataRowIndex2ExcelRow(int32(rowIndex)), "value", checkValue, "cell_value", thisValue, "target_sheet", fkSheetName, "target_field", fkFiledName)
							}
						}
					}

				} else {
					// 有内嵌结构

					// 检测外键的表是否存在
					refSheetParser := root.getSheetParser(fkSheetName)
					customParser := root.getSheetParser(embedSheetName)
					if refSheetParser == nil || customParser == nil {
						s.logger.Error("embedded foreign key target sheet not found", "field", head.Name(), "embedded_sheet", embedSheetName, "target_sheet", fkSheetName, "target_field", fkFiledName)
						continue
					}

					// 检测外键的字段值是否存在（支持repeated）
					for rowIndex, _ := range s.dataRows {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						values := SplitCustomValue(thisValue)
						for _, checkMsg := range values {
							idx := refSheetParser.getFieldColIndex(fkFiledName)
							s.checkCustomFieldCount(head, int32(rowIndex), thisValue, checkMsg, customParser)
							checkValue := checkMsg[idx]
							if !refSheetParser.hasExistFiledValue(fkFiledName, checkValue) {
								s.logger.Error("embedded foreign key value not found", "field", head.Name(), "excel_row", DataRowIndex2ExcelRow(int32(rowIndex)), "value", checkValue, "cell_value", thisValue, "target_sheet", fkSheetName, "target_field", fkFiledName)
							}
						}
					}
				}

			case TagIndexName:
				// TODO
			}
		}
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
		slog.Error("parseFromFile proto message template fail", "error", err)
		return
	}

	m := &ProtoMessageModel{
		PackageName: s.getPackageName(),
		MessageName: s.sheetName,
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
		s.logger.Error("create proto output directory failed", "output_path", outPath, "filter", s.filter, "error", err)
		return
	}

	f, err := os.Create(s.getProtoFilePath())
	if err != nil {
		s.logger.Error("create proto file failed", "output_path", s.getProtoFilePath(), "filter", s.filter, "error", err)
		return
	}

	// 执行模板,输出文件
	err = tmpl.Execute(f, m)
	if err != nil {
		_ = f.Close()
		s.logger.Error("render proto template failed", "output_path", s.getProtoFilePath(), "filter", s.filter, "error", err)
		return
	}
	if err := f.Close(); err != nil {
		s.logger.Error("close proto file failed", "output_path", s.getProtoFilePath(), "filter", s.filter, "error", err)
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
		s.logger.Error("compile generated proto for data export failed", "proto_files", protoNames, "proto_path", s.getProtoOutPath(), "filter", s.filter, "error", err)
		return
	}

	// 2. 获取消息描述符
	configMsgName := fmt.Sprintf("%v.%v", s.getPackageName(), s.sheetName)
	configMsgType, err := getMessageType(resolver, configMsgName+"Config")
	if err != nil {
		s.logger.Error("resolve config protobuf message failed", "message", configMsgName+"Config", "filter", s.filter, "error", err)
		return
	}
	recordMsgType, err := getMessageType(resolver, configMsgName)
	if err != nil {
		s.logger.Error("resolve record protobuf message failed", "message", configMsgName, "filter", s.filter, "error", err)
		return
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
					s.logger.Error("resolve nested protobuf message failed", "field", fd.Name(), "message", subMsgName, "excel_row", DataRowIndex2ExcelRow(int32(rowIdx)), "filter", s.filter, "error", err2)
					return
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
		s.logger.Error("create data output directory failed", "output_path", outPath, "filter", s.filter, "error", err)
		return
	}

	// write dataRows file
	dataFilePath := s.getDataFilePath()
	if err := os.WriteFile(dataFilePath, data, os.ModePerm); err != nil {
		s.logger.Error("write protobuf data failed", "output_path", dataFilePath, "filter", s.filter, "error", err)
	}
}

func (s *SheetParser) ExportCode(root *Parser) {
	cfg := config.Cfg.Outs[FilterFullName[s.filter]]
	tplCodePath := config.Cfg.TplCodePaths[cfg.CodeLanguage]
	codeOutPath := config.Cfg.CodeOutPaths[cfg.CodeLanguage]
	switch cfg.CodeLanguage {
	case "golang":
		moduleCode := NewGolangModuleCode(tplCodePath, codeOutPath)
		moduleCode.GenCode(root, s)
	case "csharp":
		moduleCode := NewCsharpModuleCode(tplCodePath, codeOutPath)
		moduleCode.GenCode(root, s)
	}
}

func (s *SheetParser) procBaseProtoType(root *Parser, fd Head, rowIdx int32, value string) []protoreflect.Value {
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
			pbField := customMsg.Descriptor().Fields().ByName(protoreflect.Name(subFd.Name()))
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
		return protoreflect.ValueOfInt32(ToInt32(value))
	case "int64":
		return protoreflect.ValueOfInt64(ToInt64(value))
	case "float":
		return protoreflect.ValueOfFloat32(ToFloat32(value))
	case "double":
		return protoreflect.ValueOfFloat64(ToFloat64(value))
	case "bool":
		return protoreflect.ValueOfBool(ToBool(value))
	case TimestampName:
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
				return protoreflect.ValueOfEnum(protoreflect.EnumNumber(ToInt32(value)))
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

	return protoreflect.ValueOf(nil)
}
