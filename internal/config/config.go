package config

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config stores runtime settings for Memoir.
type Config struct {
	Addr               string
	DataDir            string
	InternalBaseURL    string
	AllowedOrigins     []string
	MaxUploadBytes     int64
	MaxUploadFiles     int
	OpenAIBaseURL      string
	OpenAIAPIKey       string
	OpenAIModel        string
	OpenAIImageBaseURL string
	OpenAIImageAPIKey  string
	OpenAIImageModel   string
	GitHubOwner        string
	GitHubRepo         string
	GitHubBranch       string
	GitHubToken        string
}

// AISettings stores user-editable AI provider settings.
type AISettings struct {
	BaseURL      string `json:"baseUrl"`
	APIKey       string `json:"apiKey"`
	Model        string `json:"model"`
	ImageBaseURL string `json:"imageBaseUrl"`
	ImageAPIKey  string `json:"imageApiKey"`
	ImageModel   string `json:"imageModel"`
	Enabled      bool   `json:"enabled"`
}

// GitHubSettings stores user-editable GitHub Pages publishing settings.
type GitHubSettings struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Token  string `json:"token"`
}

// Load reads environment variables and returns a Config.
func Load() Config {
	loadDotEnv(".env")

	addr := env("PORT", "8090")
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	apiKey := os.Getenv("OPENAI_API_KEY")
	imageBaseURL := os.Getenv("OPENAI_IMAGE_BASE_URL")
	if imageBaseURL == "" {
		imageBaseURL = baseURL
	}
	imageAPIKey := os.Getenv("OPENAI_IMAGE_API_KEY")
	if imageAPIKey == "" {
		imageAPIKey = apiKey
	}
	return Config{
		Addr:               addr,
		DataDir:            env("DATA_DIR", "./data"),
		InternalBaseURL:    normalizeBaseURL(env("INTERNAL_BASE_URL", defaultInternalBaseURL(addr))),
		AllowedOrigins:     parseList(env("ALLOWED_ORIGINS", "")),
		MaxUploadBytes:     int64(envInt("MAX_UPLOAD_MB", 256)) * 1024 * 1024,
		MaxUploadFiles:     envInt("MAX_UPLOAD_FILES", 200),
		OpenAIBaseURL:      baseURL,
		OpenAIAPIKey:       apiKey,
		OpenAIModel:        env("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIImageBaseURL: imageBaseURL,
		OpenAIImageAPIKey:  imageAPIKey,
		OpenAIImageModel:   env("OPENAI_IMAGE_MODEL", "gpt-image-1.5"),
	}
}

// AISettings returns the editable AI settings for the current config.
func (c Config) AISettings() AISettings {
	return AISettings{
		BaseURL:      c.OpenAIBaseURL,
		APIKey:       c.OpenAIAPIKey,
		Model:        c.OpenAIModel,
		ImageBaseURL: c.OpenAIImageBaseURL,
		ImageAPIKey:  c.OpenAIImageAPIKey,
		ImageModel:   c.OpenAIImageModel,
		Enabled:      c.OpenAIAPIKey != "",
	}
}

// ApplyAISettings updates AI provider fields on the config.
func (c Config) ApplyAISettings(settings AISettings) Config {
	c.OpenAIBaseURL = strings.TrimSpace(settings.BaseURL)
	c.OpenAIAPIKey = strings.TrimSpace(settings.APIKey)
	c.OpenAIModel = strings.TrimSpace(settings.Model)
	if c.OpenAIModel == "" {
		c.OpenAIModel = "gpt-4o-mini"
	}
	c.OpenAIImageBaseURL = strings.TrimSpace(settings.ImageBaseURL)
	c.OpenAIImageAPIKey = strings.TrimSpace(settings.ImageAPIKey)
	c.OpenAIImageModel = strings.TrimSpace(settings.ImageModel)
	if c.OpenAIImageModel == "" {
		c.OpenAIImageModel = "gpt-image-1.5"
	}
	return c
}

// GitHubSettings returns the editable GitHub Pages settings for the current config.
func (c Config) GitHubSettings() GitHubSettings {
	branch := c.GitHubBranch
	if branch == "" {
		branch = "main"
	}
	return GitHubSettings{
		Owner:  c.GitHubOwner,
		Repo:   c.GitHubRepo,
		Branch: branch,
		Token:  c.GitHubToken,
	}
}

// ApplyGitHubSettings updates GitHub Pages fields on the config.
func (c Config) ApplyGitHubSettings(settings GitHubSettings) Config {
	c.GitHubOwner = strings.TrimSpace(settings.Owner)
	c.GitHubRepo = strings.TrimSpace(settings.Repo)
	c.GitHubBranch = strings.TrimSpace(settings.Branch)
	if c.GitHubBranch == "" {
		c.GitHubBranch = "main"
	}
	c.GitHubToken = strings.TrimSpace(settings.Token)
	return c
}

// LoadAISettings reads persisted AI settings from disk when present.
func LoadAISettings(path string) (AISettings, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return AISettings{}, false, nil
	}
	if err != nil {
		return AISettings{}, false, err
	}
	var settings AISettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return AISettings{}, false, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err == nil {
		if _, ok := fields["imageBaseUrl"]; !ok {
			settings.ImageBaseURL = settings.BaseURL
		}
		if _, ok := fields["imageApiKey"]; !ok {
			settings.ImageAPIKey = settings.APIKey
		}
	}
	return settings, true, nil
}

// SaveAISettings persists AI settings under the app data directory.
func SaveAISettings(path string, settings AISettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// LoadGitHubSettings reads persisted GitHub settings from disk when present.
func LoadGitHubSettings(path string) (GitHubSettings, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return GitHubSettings{}, false, nil
	}
	if err != nil {
		return GitHubSettings{}, false, err
	}
	var settings GitHubSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return GitHubSettings{}, false, err
	}
	if settings.Branch == "" {
		settings.Branch = "main"
	}
	return settings, true, nil
}

// SaveGitHubSettings persists GitHub settings under the app data directory.
func SaveGitHubSettings(path string, settings GitHubSettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func normalizeBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}
	return strings.TrimRight(value, "/")
}

func defaultInternalBaseURL(addr string) string {
	host := "127.0.0.1"
	port := "8090"
	if strings.HasPrefix(addr, ":") {
		if trimmed := strings.TrimPrefix(addr, ":"); trimmed != "" {
			port = trimmed
		}
		return "http://" + net.JoinHostPort(host, port)
	}
	if parsedHost, parsedPort, err := net.SplitHostPort(addr); err == nil {
		if parsedPort != "" {
			port = parsedPort
		}
		parsedHost = strings.Trim(parsedHost, "[]")
		if parsedHost != "" && parsedHost != "0.0.0.0" && parsedHost != "::" {
			host = parsedHost
		}
		return "http://" + net.JoinHostPort(host, port)
	}
	if addr = strings.TrimSpace(addr); addr != "" {
		host = addr
	}
	return "http://" + net.JoinHostPort(host, port)
}

func parseList(raw string) []string {
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func loadDotEnv(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
