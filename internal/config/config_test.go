package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	if cfg.General.RefreshInterval != 5*time.Second {
		t.Errorf("Default refresh interval should be 5s, got %v", cfg.General.RefreshInterval)
	}

	if !cfg.Providers.OpenCode.Enabled {
		t.Error("OpenCode provider should be enabled by default")
	}

	if cfg.Theme.Mode != "dark" {
		t.Errorf("Default theme mode should be 'dark', got %v", cfg.Theme.Mode)
	}

	if cfg.UI.DefaultGrouping != "type" {
		t.Errorf("Default grouping should be 'type', got %v", cfg.UI.DefaultGrouping)
	}
}

func TestLoadNonexistent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load should return default config for nonexistent file, got error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load should return default config")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	cfg := DefaultConfig()
	cfg.General.LogLevel = "debug"
	cfg.Theme.Mode = "light"

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.General.LogLevel != "debug" {
		t.Errorf("Loaded log level should be 'debug', got %v", loaded.General.LogLevel)
	}

	if loaded.Theme.Mode != "light" {
		t.Errorf("Loaded theme mode should be 'light', got %v", loaded.Theme.Mode)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath should not return empty string")
	}

	if !filepath.IsAbs(path) {
		t.Error("ConfigPath should return absolute path")
	}
}
