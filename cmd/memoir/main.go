package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"memoir/internal/ai/provider"
	"memoir/internal/app"
	"memoir/internal/config"
	"memoir/internal/httpapi"
	"memoir/internal/media"
	"memoir/internal/store"
	"memoir/internal/webassets"
)

func main() {
	cfg := config.Load()
	listenAddr := cfg.Addr
	if strings.TrimSpace(os.Getenv("PORT")) == "" {
		listenAddr = "127.0.0.1:0"
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	publicURL := publicBaseURL(listener.Addr())
	if strings.TrimSpace(os.Getenv("INTERNAL_BASE_URL")) == "" {
		cfg.InternalBaseURL = publicURL
	}

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

	webFS, err := webassets.FS()
	if err != nil {
		log.Fatalf("open embedded web UI: %v", err)
	}
	if err := validateWebUI(webFS); err != nil {
		log.Fatal(err)
	}

	service := app.NewService(projectStore, mediaManager, analyzer, cfg, provider.NewAnalyzer, settingsPath, githubSettingsPath)
	server := httpapi.NewServer(service, mediaManager, httpapi.ServerOptions{
		AllowedOrigins: cfg.AllowedOrigins,
		MaxUploadBytes: cfg.MaxUploadBytes,
		MaxUploadFiles: cfg.MaxUploadFiles,
		WebAssets:      webFS,
	})

	log.Printf("Memoir is ready: %s", publicURL)
	openBrowserSoon(publicURL)
	if err := http.Serve(listener, server.Handler()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func publicBaseURL(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String()
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func validateWebUI(webFS fs.FS) error {
	if _, err := fs.Stat(webFS, "index.html"); err != nil {
		return errors.New("embedded web UI is missing index.html; run `make package` to build a release binary")
	}
	return nil
}

func openBrowserSoon(url string) {
	if strings.TrimSpace(os.Getenv("MEMOIR_NO_BROWSER")) == "1" {
		return
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := openBrowser(url); err != nil {
			log.Printf("open browser: %v", err)
		}
	}()
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(cmd.Args, " "), err)
	}
	return nil
}
