package parser

import (
	"fmt"
	"strings"
)

const (
	TagFkName    = "fk"
	TagIndexName = "index"
)

type HeadTag string

func (t HeadTag) IsForeignKey() bool {
	return strings.HasPrefix(string(t), TagFkName+":")
}

// GetKey 获取标签名
func (t HeadTag) GetKey() string {
	ss := strings.Split(string(t), ":")
	return ss[0]
}

// ParseForeignKey 解析ref标签
// 简单结构:  fk:Item.ID;
// 内嵌结构： fk:Resource.ID=Item.ID;
func (t HeadTag) ParseForeignKey() (embedSheetName, embedFiledName string, fkSheetName, fkFiledName string, err error) {
	key, value, found := strings.Cut(string(t), ":")
	if !found || key != TagFkName || value == "" {
		return "", "", "", "", fmt.Errorf("invalid foreign key tag %q", t)
	}
	parseReference := func(reference string) (string, string, error) {
		parts := strings.Split(reference, ".")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid foreign key reference %q", reference)
		}
		return parts[0], parts[1], nil
	}

	if !strings.Contains(value, "=") {
		fkSheetName, fkFiledName, err = parseReference(value)
		return "", "", fkSheetName, fkFiledName, err
	}

	embedReference, targetReference, found := strings.Cut(value, "=")
	if !found || strings.Contains(targetReference, "=") {
		return "", "", "", "", fmt.Errorf("invalid embedded foreign key tag %q", t)
	}
	embedSheetName, embedFiledName, err = parseReference(embedReference)
	if err != nil {
		return "", "", "", "", err
	}
	fkSheetName, fkFiledName, err = parseReference(targetReference)
	return embedSheetName, embedFiledName, fkSheetName, fkFiledName, err
}

func (t HeadTag) IsIndex() bool {
	return string(t) == TagIndexName
}
