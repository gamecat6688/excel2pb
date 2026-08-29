package parser

import (
	"excel2pb/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfigRejectsOutputOverlappingInputs(t *testing.T) {
	configureTestExports(t)
	client := config.Cfg.Outs["Client"]
	client.ProtoPath = config.Cfg.ExcelDir
	config.Cfg.Outs["Client"] = client

	if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "overlaps input") {
		t.Fatalf("unexpected overlap validation result: %v", err)
	}
}

func TestValidateConfigRejectsSharedOutputRoots(t *testing.T) {
	configureTestExports(t)
	client := config.Cfg.Outs["Client"]
	server := config.Cfg.Outs["Server"]
	server.ProtoPath = client.ProtoPath
	config.Cfg.Outs["Server"] = server

	if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "overlaps output") {
		t.Fatalf("unexpected shared-output validation result: %v", err)
	}
}

func TestValidateConfigAcceptsConfiguredTestDirectories(t *testing.T) {
	dir := configureTestExports(t)
	if err := validateConfig(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if overlap, err := pathsOverlap(filepath.Join(dir, "a"), filepath.Join(dir, "ab")); err != nil || overlap {
		t.Fatalf("sibling directories considered overlapping: overlap=%v err=%v", overlap, err)
	}
}

func TestPathsOverlapTreatsDifferentVolumesAsSeparate(t *testing.T) {
	overlap, err := pathsOverlap(`C:\source`, `D:\output`)
	if err != nil || overlap {
		t.Fatalf("different volumes considered overlapping: overlap=%v err=%v", overlap, err)
	}
}

func TestValidateConfigRejectsUnsupportedProtoImportPrefix(t *testing.T) {
	configureTestExports(t)
	config.Cfg.ProtoImportPath = "shared/"
	if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "ProtoImportPath") {
		t.Fatalf("unexpected proto import prefix validation result: %v", err)
	}
}

func TestValidateConfigRequiresValidGoPackagePath(t *testing.T) {
	for _, invalid := range []string{"", "/absolute/pbs", `server\pbs`, "pbs"} {
		t.Run(invalid, func(t *testing.T) {
			configureTestExports(t)
			server := config.Cfg.Outs["Server"]
			server.GoPackagePath = invalid
			config.Cfg.Outs["Server"] = server
			if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "GoPackagePath") {
				t.Fatalf("unexpected Go package validation result: %v", err)
			}
		})
	}
}

func TestValidateConfigRequiresMatchingGoModulePath(t *testing.T) {
	for name, modulePath := range map[string]string{
		"missing": "",
		"outside": "example.com/other",
	} {
		t.Run(name, func(t *testing.T) {
			configureTestExports(t)
			server := config.Cfg.Outs["Server"]
			server.GoModulePath = modulePath
			config.Cfg.Outs["Server"] = server
			if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "GoModulePath") {
				t.Fatalf("unexpected Go module validation result: %v", err)
			}
		})
	}
}

func TestGoPackageRelativePathStripsModule(t *testing.T) {
	cfg := config.OutConfig{GoModulePath: "example.com/game/server", GoPackagePath: "example.com/game/server/pbs"}
	if got, err := goPackageRelativePath(cfg); err != nil || got != "pbs" {
		t.Fatalf("relative Go package path = %q, %v", got, err)
	}
}

func TestValidateConfigRejectsTargetLanguagePackageKeywords(t *testing.T) {
	for target, packageName := range map[string]string{"Server": "type", "Client": "class"} {
		t.Run(target, func(t *testing.T) {
			configureTestExports(t)
			out := config.Cfg.Outs[target]
			out.PackageName = packageName
			config.Cfg.Outs[target] = out
			if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("unexpected package keyword validation result: %v", err)
			}
		})
	}
}

func TestValidateConfigRejectsSymlinkedOutputIntoInput(t *testing.T) {
	dir := configureTestExports(t)
	if err := os.MkdirAll(config.Cfg.ExcelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "excel-alias")
	if err := os.Symlink(config.Cfg.ExcelDir, alias); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}
	client := config.Cfg.Outs["Client"]
	client.ProtoPath = alias
	config.Cfg.Outs["Client"] = client
	if err := validateConfig(); err == nil || !strings.Contains(err.Error(), "overlaps input") {
		t.Fatalf("unexpected symlink overlap validation result: %v", err)
	}
}
