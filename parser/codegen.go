package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// 加载代码生成器
type LoaderCodeGenerator interface {
	GenCode(root *Parser, sheet *SheetParser) bool
}

// 模块代码生成器
type ModuleCodeGenerator interface {
	GenCode(root *Parser, sheet *SheetParser) bool
}

func findCodeTemplates(templatePath string, modules bool) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(templatePath, "*.*"))
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(matches))
	for _, filename := range matches {
		isModule := strings.Contains(filepath.Base(filename), "{")
		if isModule == modules {
			filtered = append(filtered, filename)
		}
	}
	if len(filtered) == 0 {
		kind := "loader"
		if modules {
			kind = "module"
		}
		return nil, fmt.Errorf("no %s templates found in %q", kind, templatePath)
	}
	return filtered, nil
}
