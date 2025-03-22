package parser

import "strings"

const (
	TagFkName    = "fk"
	TagIndexName = "index"
)

type HeadTag string

func (t HeadTag) IsForeignKey() bool {
	return strings.Index(string(t), TagFkName) != -1
}

// GetKey 获取标签名
func (t HeadTag) GetKey() string {
	ss := strings.Split(string(t), ":")
	return ss[0]
}

// ParseForeignKey 解析ref标签
// 简单结构:  fk:Item.ID;
// 内嵌结构： fk:Resource.ID=Item.ID;
func (t HeadTag) ParseForeignKey() (embedSheetName, embedFiledName string, fkSheetName, fkFiledName string) {
	if strings.Index(string(t), "=") == -1 {
		ss := strings.Split(string(t), ":")
		nn := strings.Split(ss[1], ".")
		return "", "", nn[0], nn[1]
	}

	// 有=号，说明是子表
	ss := strings.Split(string(t), ":")
	cc := strings.Split(ss[1], "=")
	ems := strings.Split(cc[0], ".")
	fks := strings.Split(cc[1], ".")
	return ems[0], ems[1], fks[0], fks[1]
}

func (t HeadTag) IsIndex() bool {
	return strings.Index(string(t), TagIndexName) != -1
}
