package parser

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

const embeddedTemplatePrefix = "embedded://"

var embeddedTemplates fs.FS

// SetEmbeddedTemplates provides templates compiled into the command executable.
func SetEmbeddedTemplates(templateFS fs.FS) {
	embeddedTemplates = templateFS
}

// 加载代码生成器
type LoaderCodeGenerator interface {
	GenCode(root *Parser, sheet *SheetParser) bool
}

// 模块代码生成器
type ModuleCodeGenerator interface {
	GenCode(root *Parser, sheet *SheetParser) bool
}

func findCodeTemplates(templatePath string, modules bool) ([]string, error) {
	matches, err := listCodeTemplates(templatePath)
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(matches))
	for _, filename := range matches {
		isModule := strings.Contains(templateBaseName(filename), "{")
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

func listCodeTemplates(templatePath string) ([]string, error) {
	if !isEmbeddedTemplatePath(templatePath) {
		matches, err := filepath.Glob(filepath.Join(templatePath, "*.*"))
		if err != nil {
			return nil, err
		}
		sort.Strings(matches)
		return matches, nil
	}
	if embeddedTemplates == nil {
		return nil, fmt.Errorf("embedded templates are unavailable")
	}
	language := strings.Trim(strings.TrimPrefix(templatePath, embeddedTemplatePrefix), "/")
	if language == "" || !fs.ValidPath(language) || strings.Contains(language, "/") {
		return nil, fmt.Errorf("invalid embedded template path %q", templatePath)
	}
	matches, err := fs.Glob(embeddedTemplates, path.Join("assets/template", language, "*.*"))
	if err != nil {
		return nil, err
	}
	for index, filename := range matches {
		matches[index] = embeddedTemplatePrefix + filename
	}
	sort.Strings(matches)
	return matches, nil
}

func isEmbeddedTemplatePath(templatePath string) bool {
	return strings.HasPrefix(templatePath, embeddedTemplatePrefix)
}

func templateBaseName(filename string) string {
	if isEmbeddedTemplatePath(filename) {
		return path.Base(strings.TrimPrefix(filename, embeddedTemplatePrefix))
	}
	return filepath.Base(filename)
}

func parseCodeTemplate(filename string) (*template.Template, error) {
	if !isEmbeddedTemplatePath(filename) {
		return template.ParseFiles(filename)
	}
	if embeddedTemplates == nil {
		return nil, fmt.Errorf("embedded templates are unavailable")
	}
	name := strings.TrimPrefix(filename, embeddedTemplatePrefix)
	data, err := fs.ReadFile(embeddedTemplates, name)
	if err != nil {
		return nil, err
	}
	return template.New(path.Base(name)).Parse(string(data))
}
