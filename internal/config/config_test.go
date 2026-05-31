package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesAllowedOrigins(t *testing.T) {
	t.Setenv("PORT", "8123")
	t.Setenv("DATA_DIR", "/tmp/memoir-test")
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000, https://memoir.example.com ")
	t.Setenv("MAX_UPLOAD_MB", "12")
	t.Setenv("MAX_UPLOAD_FILES", "34")

	cfg := Load()

	if cfg.Addr != ":8123" {
		t.Fatalf("expected addr :8123, got %q", cfg.Addr)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %#v", cfg.AllowedOrigins)
	}
	if cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("unexpected first origin: %q", cfg.AllowedOrigins[0])
	}
	if cfg.AllowedOrigins[1] != "https://memoir.example.com" {
		t.Fatalf("unexpected second origin: %q", cfg.AllowedOrigins[1])
	}
	if cfg.MaxUploadBytes != 12*1024*1024 {
		t.Fatalf("expected max upload bytes for 12 MB, got %d", cfg.MaxUploadBytes)
	}
	if cfg.MaxUploadFiles != 34 {
		t.Fatalf("expected max upload files 34, got %d", cfg.MaxUploadFiles)
	}
	if cfg.InternalBaseURL != "http://127.0.0.1:8123" {
		t.Fatalf("expected internal base URL from port, got %q", cfg.InternalBaseURL)
	}
}

func TestLoadUsesInternalBaseURLOverride(t *testing.T) {
	t.Setenv("INTERNAL_BASE_URL", "memoir-api:8090/")

	cfg := Load()

	if cfg.InternalBaseURL != "http://memoir-api:8090" {
		t.Fatalf("expected normalized internal base URL, got %q", cfg.InternalBaseURL)
	}
}

func TestLoadUsesSeparateImageOptimizationEnv(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://vision.example/v1")
	t.Setenv("OPENAI_API_KEY", "vision-key")
	t.Setenv("OPENAI_MODEL", "vision-model")
	t.Setenv("OPENAI_IMAGE_BASE_URL", "https://image.example/v1")
	t.Setenv("OPENAI_IMAGE_API_KEY", "image-key")
	t.Setenv("OPENAI_IMAGE_MODEL", "image-model")

	cfg := Load()

	if cfg.OpenAIBaseURL != "https://vision.example/v1" || cfg.OpenAIAPIKey != "vision-key" || cfg.OpenAIModel != "vision-model" {
		t.Fatalf("unexpected vision config: %#v", cfg)
	}
	if cfg.OpenAIImageBaseURL != "https://image.example/v1" || cfg.OpenAIImageAPIKey != "image-key" || cfg.OpenAIImageModel != "image-model" {
		t.Fatalf("unexpected image optimization config: %#v", cfg)
	}
}

func TestLoadUsesVisionEnvAsInitialImageOptimizationFallback(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://vision.example/v1")
	t.Setenv("OPENAI_API_KEY", "vision-key")

	cfg := Load()

	if cfg.OpenAIImageBaseURL != cfg.OpenAIBaseURL {
		t.Fatalf("expected image base URL to fall back to vision base URL, got %q", cfg.OpenAIImageBaseURL)
	}
	if cfg.OpenAIImageAPIKey != cfg.OpenAIAPIKey {
		t.Fatalf("expected image API key to fall back to vision API key, got %q", cfg.OpenAIImageAPIKey)
	}
}

func TestLoadAISettingsBackfillsImageOptimizationFieldsForOldSettings(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ai-settings.json")
	raw := []byte(`{
  "baseUrl": "https://vision.example/v1",
  "apiKey": "vision-key",
  "model": "vision-model",
  "imageModel": "image-model",
  "enabled": true
}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	settings, ok, err := LoadAISettings(path)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if !ok {
		t.Fatalf("expected settings file to be loaded")
	}
	if settings.ImageBaseURL != settings.BaseURL || settings.ImageAPIKey != settings.APIKey {
		t.Fatalf("expected old settings to backfill image provider fields, got %#v", settings)
	}
}
