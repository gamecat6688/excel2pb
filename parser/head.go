package parser

import "strings"

const (
	I18nName       = "i18n"
	PrimaryKeyName = "pk"
	UniqueName     = "unique"
	RepeatedName   = "repeated"
	ClientFlag     = "c"
	ServerFlag     = "s"
)

var (
	AllFilters = []string{
		ClientFlag,
		ServerFlag,
	}
)

// 字段功能
const (
	// HeadName 变量名，英文命名，且需要符合导出语言的命名规则
	// 首字母大写，如: ID, Name, Level, etc.
	HeadName = iota

	// HeadType 变量类型
	// sting,int32,int64
	// unique 类型 (表示唯一性)
	// pk 类型 (表示主键，主键具有唯一性)、
	// repeated 类型 （表示数组，用 , 分割）
	HeadType

	// HeadExport 客户端或服务器导出过滤
	// c - 客户端导出, s - 服务器导出, cs - 都导出
	HeadExport // 导出过滤

	// HeadDesc 字段中文说明
	HeadDesc

	HeadCount
)

type Head struct {
	info [HeadCount]string
	tags []HeadTag // 字段标签（已拆分）
}

func (h Head) Name() string {
	return h.info[HeadName]
}

func (h Head) Desc() string {
	return h.info[HeadDesc]
}

// Type 返回类型，包含repeated, pk, unique等修饰符
func (h Head) Type() string {
	return h.info[HeadType]
}

// BaseType 只返回基础类型，不返回repeated, pk, unique等修饰符
func (h Head) BaseType() string {
	ss := strings.Split(h.Type(), " ")
	if len(ss) == 1 {
		return ss[0]
	}
	return strings.Trim(ss[1], " ")
}

// ProtoType 返回protobuf支持的类型
func (h Head) ProtoType() string {
	ss := strings.Split(h.Type(), " ")
	if ss[0] == "repeated" {
		return h.Type()
	} else if ss[0] == I18nName {
		return "string"
	}
	return h.BaseType()
}

func (h Head) HasTags() bool {
	return len(h.tags) > 0
}

// IsPrimaryKey 是否主键
func (h Head) IsPrimaryKey() bool {
	return strings.Index(h.Type(), PrimaryKeyName) != -1
}

// IsUnique 是否唯一性
func (h Head) IsUnique() bool {
	return strings.Index(h.Type(), UniqueName) != -1
}

// IsRepeated 是否数组
func (h Head) IsRepeated() bool {
	return strings.Index(h.Type(), RepeatedName) != -1
}

// IsI18n 是否多语言字段
func (h Head) IsI18n() bool {
	return strings.Index(h.Type(), I18nName) != -1
}

func (h Head) IsFilter(key string) bool {
	str := h.info[HeadExport]
	if str == "" {
		return false
	}

	return strings.Index(str, key) != -1
}

// IsExportClient 是否导出客户端 C
func (h Head) IsExportClient() bool {
	return h.IsFilter(ClientFlag)
}

// IsExportServer 是否导出服务器 S
func (h Head) IsExportServer() bool {
	return h.IsFilter(ServerFlag)
}
