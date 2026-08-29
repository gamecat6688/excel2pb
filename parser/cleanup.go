package parser

import (
	"excel2pb/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const generatedFilesManifest = ".excel2pb-generated.json"

type managedPattern struct {
	pattern   string
	recursive bool
}

type managedOutput struct {
	root     string
	expected map[string]struct{}
	patterns []managedPattern
}

func (p *Parser) cleanupStaleOutputs() {
	outputs, err := p.managedOutputs()
	if err != nil {
		panic(fmt.Sprintf("build generated output manifest failed: %v", err))
	}
	for _, output := range outputs {
		if err := output.cleanup(); err != nil {
			panic(fmt.Sprintf("clean stale generated files in %q failed: %v", output.root, err))
		}
	}
}

func (p *Parser) managedOutputs() (map[string]*managedOutput, error) {
	outputs := map[string]*managedOutput{}
	add := func(root, relative string) error {
		output, err := getManagedOutput(outputs, root)
		if err != nil {
			return err
		}
		path, err := output.safePath(relative)
		if err != nil {
			return err
		}
		relative, err = filepath.Rel(output.root, path)
		if err != nil {
			return err
		}
		output.expected[filepath.Clean(relative)] = struct{}{}
		return nil
	}
	addPattern := func(root, pattern string, recursive bool) error {
		output, err := getManagedOutput(outputs, root)
		if err != nil {
			return err
		}
		for _, existing := range output.patterns {
			if existing.pattern == pattern && existing.recursive == recursive {
				return nil
			}
		}
		output.patterns = append(output.patterns, managedPattern{pattern: pattern, recursive: recursive})
		return nil
	}

	protoNames := make([]string, 0, len(p.sheets)+len(p.enums))
	for name := range p.sheets {
		protoNames = append(protoNames, name)
	}
	for name := range p.enums {
		protoNames = append(protoNames, name)
	}

	for _, filter := range AllFilters {
		cfg := config.Cfg.Outs[FilterFullName[filter]]
		if cfg.DataExt == "" {
			return nil, fmt.Errorf("missing data extension for filter %q", filter)
		}
		if err := addPattern(cfg.ProtoPath, "*.proto", false); err != nil {
			return nil, err
		}
		if err := addPattern(cfg.DataPath, "*"+cfg.DataExt, false); err != nil {
			return nil, err
		}
		for _, name := range protoNames {
			if err := add(cfg.ProtoPath, name+".proto"); err != nil {
				return nil, err
			}
		}
		for name, sheet := range p.sheets {
			if sheet.HasData() {
				if err := add(cfg.DataPath, name+cfg.DataExt); err != nil {
					return nil, err
				}
			}
		}

		switch cfg.CodeLanguage {
		case "csharp":
			if err := addPattern(cfg.PbPath, "*.cs", true); err != nil {
				return nil, err
			}
			for _, name := range protoNames {
				if err := add(cfg.PbPath, csharpProtoFilename(name)); err != nil {
					return nil, err
				}
			}
		case "golang":
			if err := addPattern(cfg.PbPath, "*.pb.go", true); err != nil {
				return nil, err
			}
			packageDirectory, err := goPackageRelativePath(cfg)
			if err != nil {
				return nil, err
			}
			for _, name := range protoNames {
				if err := add(cfg.PbPath, filepath.Join(packageDirectory, name+".pb.go")); err != nil {
					return nil, err
				}
			}
		case "godot":
			if err := addPattern(cfg.PbPath, "*.proto.gd", true); err != nil {
				return nil, err
			}
			for _, name := range protoNames {
				if err := add(cfg.PbPath, name+".proto.gd"); err != nil {
					return nil, err
				}
			}
		}

		if err := addCodeOutputs(outputs, cfg.CodeLanguage, p.sheets); err != nil {
			return nil, err
		}
	}

	return outputs, nil
}

func csharpProtoFilename(protoBaseName string) string {
	var result strings.Builder
	uppercaseNext := true
	for _, char := range protoBaseName {
		if char == '_' {
			uppercaseNext = true
			continue
		}
		if uppercaseNext {
			char = unicode.ToUpper(char)
			uppercaseNext = false
		}
		result.WriteRune(char)
	}
	return result.String() + ".cs"
}

func addCodeOutputs(outputs map[string]*managedOutput, language string, sheets map[string]*SheetParser) error {
	templateRoot := config.Cfg.TplCodePaths[language]
	outputRoot := config.Cfg.CodeOutPaths[language]
	if templateRoot == "" || outputRoot == "" {
		return fmt.Errorf("missing template or code output path for language %q", language)
	}
	output, err := getManagedOutput(outputs, outputRoot)
	if err != nil {
		return err
	}
	templates, err := listCodeTemplates(templateRoot)
	if err != nil {
		return err
	}
	for _, templatePath := range templates {
		base := templateBaseName(templatePath)
		if !strings.Contains(base, "{") {
			output.expected[base] = struct{}{}
			output.patterns = append(output.patterns, managedPattern{pattern: base})
			continue
		}

		pattern := strings.NewReplacer("{name}", "*", "{Name}", "*").Replace(base)
		output.patterns = append(output.patterns, managedPattern{pattern: pattern})
		for name, sheet := range sheets {
			if !sheet.HasData() {
				continue
			}
			generated := strings.NewReplacer("{name}", strings.ToLower(name), "{Name}", name).Replace(base)
			output.expected[generated] = struct{}{}
		}
	}
	return nil
}

func getManagedOutput(outputs map[string]*managedOutput, root string) (*managedOutput, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("managed output directory is empty")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absRoot = filepath.Clean(absRoot)
	absRoot, err = resolveExistingPath(absRoot)
	if err != nil {
		return nil, err
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	protectedRoots := []string{workingDirectory, os.TempDir()}
	if userHome, homeErr := os.UserHomeDir(); homeErr == nil {
		protectedRoots = append(protectedRoots, userHome)
	}
	for _, protected := range protectedRoots {
		resolvedProtected, resolveErr := resolveExistingPath(protected)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if pathContains(absRoot, resolvedProtected) {
			return nil, fmt.Errorf("refuse to manage broad output directory %q", absRoot)
		}
	}
	if absRoot == filepath.VolumeName(absRoot)+string(filepath.Separator) {
		return nil, fmt.Errorf("refuse to manage broad output directory %q", absRoot)
	}
	output := outputs[absRoot]
	if output == nil {
		output = &managedOutput{root: absRoot, expected: map[string]struct{}{}}
		outputs[absRoot] = output
	}
	return output, nil
}

func resolveExistingPath(path string) (string, error) {
	path = filepath.Clean(path)
	existing := path
	var suffix []string
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return path, nil
		}
		suffix = append(suffix, filepath.Base(existing))
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, suffix[i])
	}
	return filepath.Clean(resolved), nil
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func (o *managedOutput) cleanup() error {
	if err := os.MkdirAll(o.root, 0o755); err != nil {
		return err
	}
	candidates := map[string]struct{}{}
	manifestPath := filepath.Join(o.root, generatedFilesManifest)
	// Remove manifests produced by early versions of the cleanup implementation.
	// Delivery directories should contain only the requested artifacts.
	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	err := filepath.WalkDir(o.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path == manifestPath {
			return nil
		}
		relative, err := filepath.Rel(o.root, path)
		if err != nil {
			return err
		}
		for _, managed := range o.patterns {
			if !managed.recursive && filepath.Dir(relative) != "." {
				continue
			}
			matched, err := filepath.Match(managed.pattern, filepath.Base(relative))
			if err != nil {
				return err
			}
			if matched {
				candidates[filepath.Clean(relative)] = struct{}{}
				break
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for relative := range candidates {
		if _, keep := o.expected[relative]; keep {
			continue
		}
		path, err := o.safePath(relative)
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func (o *managedOutput) safePath(relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("manifest contains absolute path %q", relative)
	}
	path := filepath.Clean(filepath.Join(o.root, relative))
	rel, err := filepath.Rel(o.root, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("manifest path %q escapes output directory", relative)
	}
	return path, nil
}
