package parser

import (
	"excel2pb/config"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type configuredPath struct {
	name string
	path string
}

var goReservedIdentifiers = map[string]struct{}{
	"break": {}, "default": {}, "func": {}, "interface": {}, "select": {}, "case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
	"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {}, "const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
	"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
}

var csharpReservedIdentifiers = map[string]struct{}{
	"abstract": {}, "as": {}, "base": {}, "bool": {}, "break": {}, "byte": {}, "case": {}, "catch": {}, "char": {}, "checked": {},
	"class": {}, "const": {}, "continue": {}, "decimal": {}, "default": {}, "delegate": {}, "do": {}, "double": {}, "else": {}, "enum": {},
	"event": {}, "explicit": {}, "extern": {}, "false": {}, "finally": {}, "fixed": {}, "float": {}, "for": {}, "foreach": {}, "goto": {},
	"if": {}, "implicit": {}, "in": {}, "int": {}, "interface": {}, "internal": {}, "is": {}, "lock": {}, "long": {}, "namespace": {},
	"new": {}, "null": {}, "object": {}, "operator": {}, "out": {}, "override": {}, "params": {}, "private": {}, "protected": {},
	"public": {}, "readonly": {}, "ref": {}, "return": {}, "sbyte": {}, "sealed": {}, "short": {}, "sizeof": {}, "stackalloc": {},
	"static": {}, "string": {}, "struct": {}, "switch": {}, "this": {}, "throw": {}, "true": {}, "try": {}, "typeof": {}, "uint": {},
	"ulong": {}, "unchecked": {}, "unsafe": {}, "ushort": {}, "using": {}, "virtual": {}, "void": {}, "volatile": {}, "while": {},
}

func validateConfig() error {
	if config.Cfg == nil {
		return fmt.Errorf("configuration is not loaded")
	}
	if strings.TrimSpace(config.Cfg.ExcelDir) == "" {
		return fmt.Errorf("ExcelDir is empty")
	}
	if config.Cfg.MaxProcess < 0 {
		return fmt.Errorf("MaxProcess must be non-negative")
	}
	if _, err := time.Parse("Z07:00", config.Cfg.TimeZone); err != nil {
		return fmt.Errorf("invalid TimeZone %q: %w", config.Cfg.TimeZone, err)
	}
	if strings.TrimSpace(config.Cfg.ProtoImportPath) != "" {
		return fmt.Errorf("ProtoImportPath is not supported; generated proto imports are relative and this value must be empty")
	}

	inputs := []configuredPath{{name: "ExcelDir", path: config.Cfg.ExcelDir}}
	for language, path := range config.Cfg.TplCodePaths {
		if strings.TrimSpace(path) != "" && !isEmbeddedTemplatePath(path) {
			inputs = append(inputs, configuredPath{name: "TplCodePaths." + language, path: path})
		}
	}

	var outputs []configuredPath
	for _, filter := range AllFilters {
		fullName := FilterFullName[filter]
		cfg, exists := config.Cfg.Outs[fullName]
		if !exists {
			return fmt.Errorf("Outs.%s is missing", fullName)
		}
		if !isValidProtobufIdentifier(cfg.PackageName) {
			return fmt.Errorf("Outs.%s.PackageName %q is not a valid protobuf identifier", fullName, cfg.PackageName)
		}
		switch cfg.CodeLanguage {
		case "golang", "csharp", "godot":
		default:
			return fmt.Errorf("Outs.%s.CodeLanguage %q is unsupported", fullName, cfg.CodeLanguage)
		}
		if cfg.CodeLanguage == "golang" {
			if _, reserved := goReservedIdentifiers[cfg.PackageName]; reserved {
				return fmt.Errorf("Outs.%s.PackageName %q is a reserved Go identifier", fullName, cfg.PackageName)
			}
		}
		if cfg.CodeLanguage == "csharp" {
			if _, reserved := csharpReservedIdentifiers[cfg.PackageName]; reserved {
				return fmt.Errorf("Outs.%s.PackageName %q is a reserved C# identifier", fullName, cfg.PackageName)
			}
		}
		if cfg.CodeLanguage == "golang" && !validGoPackagePath(cfg.GoPackagePath) {
			return fmt.Errorf("Outs.%s.GoPackagePath %q is not a valid Go import path", fullName, cfg.GoPackagePath)
		}
		if cfg.CodeLanguage == "golang" {
			if !validGoModulePath(cfg.GoModulePath) {
				return fmt.Errorf("Outs.%s.GoModulePath %q is not a valid Go module path", fullName, cfg.GoModulePath)
			}
			if cfg.GoPackagePath != cfg.GoModulePath && !strings.HasPrefix(cfg.GoPackagePath, cfg.GoModulePath+"/") {
				return fmt.Errorf("Outs.%s.GoPackagePath %q must be inside GoModulePath %q", fullName, cfg.GoPackagePath, cfg.GoModulePath)
			}
		}
		if !validDataExtension(cfg.DataExt) {
			return fmt.Errorf("Outs.%s.DataExt %q must be a simple extension beginning with a dot", fullName, cfg.DataExt)
		}
		codeOutput := config.Cfg.CodeOutPaths[cfg.CodeLanguage]
		templateRoot := config.Cfg.TplCodePaths[cfg.CodeLanguage]
		if strings.TrimSpace(codeOutput) == "" || strings.TrimSpace(templateRoot) == "" {
			return fmt.Errorf("missing code output or template path for language %q", cfg.CodeLanguage)
		}
		outputs = append(outputs,
			configuredPath{name: "Outs." + fullName + ".ProtoPath", path: cfg.ProtoPath},
			configuredPath{name: "Outs." + fullName + ".PbPath", path: cfg.PbPath},
			configuredPath{name: "Outs." + fullName + ".DataPath", path: cfg.DataPath},
			configuredPath{name: "CodeOutPaths." + cfg.CodeLanguage, path: codeOutput},
		)
	}

	for i, output := range outputs {
		if _, err := getManagedOutput(map[string]*managedOutput{}, output.path); err != nil {
			return fmt.Errorf("%s: %w", output.name, err)
		}
		for _, input := range inputs {
			overlaps, err := pathsOverlap(output.path, input.path)
			if err != nil {
				return err
			}
			if overlaps {
				return fmt.Errorf("output %s (%q) overlaps input %s (%q)", output.name, output.path, input.name, input.path)
			}
		}
		for _, previous := range outputs[:i] {
			overlaps, err := pathsOverlap(output.path, previous.path)
			if err != nil {
				return err
			}
			if overlaps {
				return fmt.Errorf("output %s (%q) overlaps output %s (%q)", output.name, output.path, previous.name, previous.path)
			}
		}
	}
	return nil
}

func validGoPackagePath(value string) bool {
	return validGoImportPath(value, true)
}

func validGoModulePath(value string) bool {
	return validGoImportPath(value, false)
}

func validGoImportPath(value string, requireQualifier bool) bool {
	if value == "" || strings.TrimSpace(value) != value || path.IsAbs(value) || path.Clean(value) != value || strings.Contains(value, `\`) {
		return false
	}
	if requireQualifier && !strings.ContainsAny(value, "./") {
		return false
	}
	for _, char := range value {
		if unicode.IsSpace(char) || unicode.IsControl(char) || strings.ContainsRune(`!"#$%&'()*,:;<=>?[\]^`+"`"+`{|}`, char) {
			return false
		}
	}
	return true
}

func goPackageRelativePath(cfg config.OutConfig) (string, error) {
	if cfg.GoPackagePath == cfg.GoModulePath {
		return "", nil
	}
	prefix := cfg.GoModulePath + "/"
	if !strings.HasPrefix(cfg.GoPackagePath, prefix) {
		return "", fmt.Errorf("GoPackagePath %q must be inside GoModulePath %q", cfg.GoPackagePath, cfg.GoModulePath)
	}
	return filepath.FromSlash(strings.TrimPrefix(cfg.GoPackagePath, prefix)), nil
}

func validDataExtension(extension string) bool {
	return strings.HasPrefix(extension, ".") && len(extension) > 1 &&
		filepath.Base(extension) == extension && !strings.ContainsAny(extension, `*?[]/\\`)
}

func pathsOverlap(first, second string) (bool, error) {
	firstAbs, err := filepath.Abs(first)
	if err != nil {
		return false, err
	}
	firstAbs, err = resolveExistingPath(firstAbs)
	if err != nil {
		return false, err
	}
	secondAbs, err := filepath.Abs(second)
	if err != nil {
		return false, err
	}
	secondAbs, err = resolveExistingPath(secondAbs)
	if err != nil {
		return false, err
	}
	firstVolume := filepath.VolumeName(firstAbs)
	secondVolume := filepath.VolumeName(secondAbs)
	if firstVolume != "" && secondVolume != "" && !strings.EqualFold(firstVolume, secondVolume) {
		return false, nil
	}
	inside := func(path, root string) (bool, error) {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return false, err
		}
		return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))), nil
	}
	firstInsideSecond, err := inside(firstAbs, secondAbs)
	if err != nil {
		return false, err
	}
	secondInsideFirst, err := inside(secondAbs, firstAbs)
	if err != nil {
		return false, err
	}
	return firstInsideSecond || secondInsideFirst, nil
}
