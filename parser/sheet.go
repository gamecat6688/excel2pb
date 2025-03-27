package parser

import (
	"excel2pb/config"
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/xuri/excelize/v2"
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

// GetFieldColIndex 获得指定字段名的列索引
func (s *SheetParser) GetFieldColIndex(name string) int32 {
	return s.headersIndexes[name]
}

// GetFieldColIndex 获得指定字段名的列索引
func (s *SheetParser) GetField(colIndex int32) Head {
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
					slog.Error("not implemented tag checker", "tag", tag)
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
			colIndex := s.GetFieldColIndex(head.Name())
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
		if root.hasSheetParser(baseType) {
			_, ok := exists[baseType]
			if !ok {
				rv = append(rv, baseType)
				exists[baseType] = true
			}
		}
	}

	return
}

// import "playerstate/ExampleNonBasicPart.proto";
func (s *SheetParser) ExportProto(root *Parser) {
	// 解析模板
	tmpl, err := template.New("proto").Parse(ProtoTemplate)
	if err != nil {
		slog.Error("parse proto template fail", "error", err)
		return
	}

	m := &ProtoModel{
		PackageName: s.getPackageName(),
		SheetName:   s.sheetName,
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
			FieldTag:  t + 1,
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

func (s *SheetParser) ExportData(root *Parser) {
	if !s.HasData() {
		return
	}

	// 获得当前表格所有的proto文件，包含依赖文件
	var protoNames []string
	protoNames = append(protoNames, s.getProtoName())
	importMessages := s.getImportMessages(root)
	for _, v := range importMessages {
		protoNames = append(protoNames, v+".proto")
	}

	// 1. 解析 .proto 文件
	parser := protoparse.Parser{}
	parser.ImportPaths = []string{
		s.getProtoOutPath(),
	}
	fileDescriptors, err := parser.ParseFiles(protoNames...)
	if err != nil {
		panic(err)
	}

	// 2. 获取消息描述符
	configMsgName := fmt.Sprintf("%v.%v", s.getPackageName(), s.sheetName)
	configDesc := fileDescriptors[0].FindMessage(configMsgName + "Config")
	recordDesc := fileDescriptors[0].FindMessage(configMsgName)
	//resourceDesc := fileDescriptors[1].FindMessage(configMsgName)
	//_ = resourceDesc

	// 5. 构造多个 record 数据
	var records []*dynamic.Message
	for _, row := range s.data {
		record := dynamic.NewMessage(recordDesc)

		for colIdx, value := range row {
			fd := s.GetField(int32(colIdx))

			if fd.IsRepeated() {
				// 数组
				if fd.IsCustom(root) {
					// TODO
					root = root
				} else {
					var arrValue []interface{}
					for _, v := range SplitBaseValue(value) {
						arrValue = append(arrValue, TypeNameToValue(fd.BaseType(), v))
					}
					record.SetFieldByName(fd.Name(), arrValue)
				}
			} else {
				// 变量
				if fd.IsCustom(root) {
					// TODO
					root = root
				} else {
					record.SetFieldByName(fd.Name(), TypeNameToValue(fd.BaseType(), value))
				}
			}

			// 构造 Cost（多个 Resource）
			//var resources []*dynamic.Message
			//for j := 0; j < 2; j++ {
			//	resource := dynamic.NewMessage(resourceDesc)
			//	resource.SetFieldByName("Id", int32(j+1))
			//	resource.SetFieldByName("Amount", int32((j+1)*100))
			//	resources = append(resources, resource)
			//}
			//record.SetFieldByName("Cost", resources)
		}

		records = append(records, record)
	}

	//msgDesc := fileDescriptors[0].FindMessage(messageName)
	//if len(fileDescriptors) > 1 {
	//	msgDesc = fileDescriptors[1].FindMessage("pb.Resource")
	//}

	// 3. 创建动态消息并填充数据
	configMsg := dynamic.NewMessage(configDesc)
	//dynamicMsg.SetFieldByName("name", "John")
	//dynamicMsg.SetFieldByName("age", int32(30))

	configMsg.SetFieldByName("Records", records)

	// 4. 序列化
	data, err := configMsg.Marshal()
	if err != nil {
		panic(err)
	}

	outPath := s.getDataOutPath()
	os.MkdirAll(outPath, os.ModePerm)

	dataFilePath := s.getDataFilePath()
	os.WriteFile(dataFilePath, data, os.ModePerm)
}

func (s *SheetParser) buildDatasetDescriptor() (*desc.MessageDescriptor, error) {
	// 1. 解析 .proto 文件
	parser := protoparse.Parser{}
	fileDescriptors, err := parser.ParseFiles("person.proto")
	if err != nil {
		panic(err)
	}

	// 2. 获取消息描述符
	msgDesc := fileDescriptors[0].FindMessage("pbs.Resource")

	// 3. 创建动态消息并填充数据
	dynamicMsg := dynamic.NewMessage(msgDesc)
	dynamicMsg.SetFieldByName("name", "John")
	dynamicMsg.SetFieldByName("age", int32(30))

	// 4. 序列化
	data, err := dynamicMsg.Marshal()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Serialized data: %x\n", data)

	return nil, nil
}
