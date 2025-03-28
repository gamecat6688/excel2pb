package parser

import (
	"excel2pb/config"
	"fmt"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
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

func (s *Sheet) IsClient() bool {
	return s.filter == ClientFlag
}

func (s *Sheet) IsServer() bool {
	return s.filter == ServerFlag
}

func (s *Sheet) HasData() bool {
	return len(s.data) > 0
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

// getFieldColIndex 获得指定字段名的列索引
func (s *SheetParser) getFieldColIndex(name string) int32 {
	return s.headersIndexes[name]
}

// GetFiled 获得指定字段名的列索引
func (s *SheetParser) GetFiled(colIndex int32) Head {
	return s.headers[colIndex]
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
	colIndex := s.getFieldColIndex(filedName)
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
	colIndex := s.getFieldColIndex(filedName)
	for _, row := range s.data {
		value := row[colIndex]
		if value == filedValue {
			return true
		}
	}
	return false
}

func (s *SheetParser) getFiledValue(filedName string, rowIndex int32) string {
	colIndex := s.getFieldColIndex(filedName)
	if int(colIndex) >= len(s.data[rowIndex]) {
		return ""
	}
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
				_ = embedFiledName

				if len(embedSheetName) == 0 {
					// 没有内嵌结构

					// 检测外键的表是否存在
					if !root.hasSheetParser(fkSheetName) {
						slog.Error(fmt.Sprintf("[%v.%v]check foreign key fail, sheet not found: %v", s.sheetName, head.Name(), fkSheetName))
					}

					// 检测外键的字段值是否存在（支持repeated）
					refSheetParser := root.getSheetParser(fkSheetName)
					for rowIndex, _ := range s.data {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						values := SplitBaseValue(thisValue)
						for _, checkValue := range values {
							if !refSheetParser.hasExistFiledValue(fkFiledName, checkValue) {
								slog.Error(fmt.Sprintf("[%v.%v]check foreign key fail, ref value is excel row=%v, failValue=%v, value=%v, not found: %v.%v",
									s.sheetName, head.Name(),
									DataRowIndex2ExcelRow(int32(rowIndex)), checkValue, thisValue, fkSheetName, fkFiledName))
							}
						}
					}

				} else {
					// 有内嵌结构

					// 检测外键的表是否存在
					if !root.hasSheetParser(fkSheetName) {
						slog.Error(fmt.Sprintf("[%v.%v]check foreign key fail, sheet not found: %v", s.sheetName, head.Name(), fkSheetName))
					}

					// 检测外键的字段值是否存在（支持repeated）
					refSheetParser := root.getSheetParser(fkSheetName)
					for rowIndex, _ := range s.data {
						thisValue := s.getFiledValue(head.Name(), int32(rowIndex))
						values := SplitCustomValue(thisValue)
						for _, checkMsg := range values {
							idx := refSheetParser.getFieldColIndex(fkFiledName)
							checkValue := checkMsg[idx]
							if !refSheetParser.hasExistFiledValue(fkFiledName, checkValue) {
								slog.Error(fmt.Sprintf("[%v.%v]check foreign key fail, ref value is excel row=%v, failValue=%v, value=%v, not found: %v.%v",
									s.sheetName, head.Name(),
									DataRowIndex2ExcelRow(int32(rowIndex)), checkValue, thisValue, fkSheetName, fkFiledName))
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
	ns.filter = filterName

	// 复制过滤后的表头
	for i, head := range s.headers {
		if head.IsFilter(filterName) {
			ns.headers = append(ns.headers, head)
			ns.headersIndexes[head.Name()] = int32(i)
		}
	}

	// 复制过滤后的数据
	for _, row := range s.data {
		var newRow []string
		for _, head := range ns.headers {
			colIndex := s.getFieldColIndex(head.Name())
			if int(colIndex) >= len(row) {
				continue
			}
			newRow = append(newRow, row[colIndex])
		}
		ns.data = append(ns.data, newRow)
	}

	// 重建表头索引
	for i, head := range ns.headers {
		ns.headersIndexes[head.Name()] = int32(i)
	}

	return ns
}

func (s *SheetParser) getPackageName() string {
	return config.ProtoPackages[s.filter]
}

func (s *SheetParser) getProtoOutPath() string {
	return config.ProtoOutPaths[s.filter]
}

func (s *SheetParser) getDataOutPath() string {
	return config.DataOutPaths[s.filter]
}

func (s *SheetParser) getDataExtension() string {
	return config.DataExtensions[s.filter]
}

func (s *SheetParser) getPbOutPath() string {
	return config.PbOutPaths[s.filter]
}

func (s *SheetParser) getGenerateLanguage() string {
	return config.GenerateLanguage[s.filter]
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
		slog.Error("parse proto message template fail", "error", err)
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
			ProtoPath: fmt.Sprintf(`"%v%v.proto";`, config.ProtoImportPath, v),
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
	os.MkdirAll(outPath, os.ModePerm)

	f, err := os.Create(s.getProtoFilePath())
	if err != nil {
		slog.Error("create proto file fail", "error", err)
		return
	}

	// 执行模板,输出文件
	err = tmpl.Execute(f, m)
	if err != nil {
		slog.Error("tmpl.Execute fail", "error", err)
		return
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
//	switch s.getGenerateLanguage() {
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
		log.Fatal(err)
	}

	// 构建一个索引map，key is proto name
	//indexesFileDescriptors := make(map[string]*desc.FileDescriptor)
	//for _, v := range fileDescriptors {
	//	indexesFileDescriptors[v.GetName()] = v
	//}

	// 2. 获取消息描述符
	configMsgName := fmt.Sprintf("%v.%v", s.getPackageName(), s.sheetName)
	configMsgType, err := getMessageType(resolver, configMsgName+"Config")
	if err != nil {
		log.Fatal(err)
	}
	recordMsgType, err := getMessageType(resolver, configMsgName)
	if err != nil {
		log.Fatal(err)
	}

	// 5. 构造多个 record 数据
	var records []protoreflect.Message
	for _, row := range s.data {
		record := recordMsgType.New()

		for colIdx, value := range row {
			fd := s.GetFiled(int32(colIdx))
			pbField := record.Descriptor().Fields().ByName(protoreflect.Name(fd.Name()))
			if fd.IsCustomMessage(root) {
				// message字段
				subMsgName := fmt.Sprintf("%v.%v", s.getPackageName(), fd.BaseType())
				subMsgType, err2 := getMessageType(resolver, subMsgName)
				if err2 != nil {
					log.Fatal(err2)
				}

				customs := s.procCustomProtoType(root, subMsgType, fd, value)
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
				arrValues := s.procBaseProtoType(root, fd, value)
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
		panic(err)
	}

	// make dirs
	outPath := s.getDataOutPath()
	os.MkdirAll(outPath, os.ModePerm)

	// write data file
	dataFilePath := s.getDataFilePath()
	os.WriteFile(dataFilePath, data, os.ModePerm)
}

func (s *SheetParser) procBaseProtoType(root *Parser, fd Head, value string) []protoreflect.Value {
	var arrValue []protoreflect.Value
	for _, v := range SplitBaseValue(value) {
		arrValue = append(arrValue, TypeNameToValue(root, s.sheetName, fd.Name(), fd.BaseType(), v))
	}
	return arrValue
}

func (s *SheetParser) procCustomProtoType(root *Parser, msgType protoreflect.MessageType, fd Head, value string) []protoreflect.Value {
	var customs []protoreflect.Value

	customParser := root.getSheetParser(fd.BaseType())

	// 构造多个子结构
	customRows := SplitCustomValue(value)
	for _, customRow := range customRows {
		customMsg := msgType.New()
		for idx, subValue := range customRow {
			subFd := customParser.GetFiled(int32(idx))
			pbField := customMsg.Descriptor().Fields().ByName(protoreflect.Name(subFd.Name()))
			customMsg.Set(pbField, TypeNameToValue(root, s.sheetName, subFd.Name(), subFd.BaseType(), subValue))
		}
		customs = append(customs, protoreflect.ValueOfMessage(customMsg))
	}

	return customs
}
