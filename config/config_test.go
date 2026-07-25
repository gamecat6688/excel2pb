package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromData(t *testing.T) {
	cfg, err := loadConfigFromData([]byte("TimeZone: '+00:00'\nOuts:\n  Client:\n    PackageName: pb\n"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.TimeZone != "+00:00" || cfg.Outs["Client"].PackageName != "pb" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if _, err := loadConfigFromData([]byte("Outs: [")); err == nil {
		t.Fatal("invalid YAML must return an error")
	}
}

func TestLoadConfigReadsFileAndFallsBackToDefaults(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	originalCfg := Cfg
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
		Cfg = originalCfg
	})

	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("TimeZone: '+09:00'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	LoadConfig()
	if Cfg.TimeZone != "+09:00" {
		t.Fatalf("file config timezone = %q", Cfg.TimeZone)
	}
	if err := os.Remove(filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	LoadConfig()
	if Cfg.TimeZone != "+08:00" || Cfg.Outs["Server"].CodeLanguage != "golang" {
		t.Fatalf("default config was not loaded: %#v", Cfg)
	}
}
