package provider

import (
	"errors"

	"memoir/internal/ai"
	"memoir/internal/ai/eino"
	"memoir/internal/config"
)

// NewAnalyzer creates an Eino-backed analyzer. Returns an error if no API key is configured.
func NewAnalyzer(cfg config.Config) (ai.Analyzer, error) {
	if cfg.OpenAIAPIKey == "" {
		return nil, errors.New("需要配置 OpenAI API Key 才能使用分析功能，请在设置中配置 OPENAI_API_KEY")
	}
	return eino.New(cfg)
}
