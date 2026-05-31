package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"memoir/internal/ai/provider"
	"memoir/internal/app"
	"memoir/internal/config"
	"memoir/internal/httpapi"
	"memoir/internal/media"
	"memoir/internal/store"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	settingsPath := filepath.Join(cfg.DataDir, "ai-settings.json")
	if settings, ok, err := config.LoadAISettings(settingsPath); err != nil {
		log.Fatalf("load ai settings: %v", err)
	} else if ok {
		cfg = cfg.ApplyAISettings(settings)
	}

	githubSettingsPath := filepath.Join(cfg.DataDir, "github-settings.json")
	if settings, ok, err := config.LoadGitHubSettings(githubSettingsPath); err != nil {
		log.Fatalf("load github settings: %v", err)
	} else if ok {
		cfg = cfg.ApplyGitHubSettings(settings)
	}

	projectStore, err := store.NewFileStore(filepath.Join(cfg.DataDir, "state.json"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	mediaManager, err := media.NewManager(filepath.Join(cfg.DataDir, "media"))
	if err != nil {
		log.Fatalf("open media manager: %v", err)
	}

	analyzer, err := provider.NewAnalyzer(cfg)
	if err != nil {
		log.Printf("WARNING: AI analyzer not available: %v. Analysis features will be disabled until an API key is configured.", err)
	}

	service := app.NewService(projectStore, mediaManager, analyzer, cfg, provider.NewAnalyzer, settingsPath, githubSettingsPath)
	server := httpapi.NewServer(service, mediaManager, httpapi.ServerOptions{
		AllowedOrigins: cfg.AllowedOrigins,
		MaxUploadBytes: cfg.MaxUploadBytes,
		MaxUploadFiles: cfg.MaxUploadFiles,
	})

	log.Printf("memoir api listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, server.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
