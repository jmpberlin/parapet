package llm

import (
	"fmt"
	"strings"

	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/anthropic"
	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/openai"
	"github.com/jmpberlin/nightwatch/backend/internal/usecase"
)

// Environment variables that configure vulnerability extraction:
//
//	LLM_PROVIDER  required. Which API standard to speak:
//	                "anthropic" – Anthropic Messages API (Claude models)
//	                "openai"    – OpenAI Chat Completions API, i.e. OpenAI itself or
//	                              any compatible provider such as TensorX
//	LLM_API_KEY   required. API key of the selected provider.
//	LLM_MODEL     required. Model id as the provider names it, e.g.
//	                anthropic: "claude-haiku-4-5"
//	                openai:    "z-ai/glm-5.3-flash" (TensorX)
//	LLM_BASE_URL  API root including the version segment.
//	                openai:    required, e.g. "https://api.openai.com/v1",
//	                           "https://api.tensorx.ai/v1"
//	                anthropic: optional, defaults to the official Anthropic API
const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
)

type Config struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

func (c Config) LogFields() []any {
	return []any{"provider", c.Provider, "model", c.Model, "base_url", c.BaseURL}
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		Provider: strings.ToLower(strings.TrimSpace(getenv("LLM_PROVIDER"))),
		APIKey:   getenv("LLM_API_KEY"),
		BaseURL:  getenv("LLM_BASE_URL"),
		Model:    getenv("LLM_MODEL"),
	}

	if cfg.Provider != ProviderAnthropic && cfg.Provider != ProviderOpenAI {
		return cfg, fmt.Errorf("LLM_PROVIDER must be %q or %q, got %q", ProviderAnthropic, ProviderOpenAI, cfg.Provider)
	}

	var missing []string
	if cfg.APIKey == "" {
		missing = append(missing, "LLM_API_KEY")
	}
	if cfg.Model == "" {
		missing = append(missing, "LLM_MODEL")
	}
	if cfg.Provider == ProviderOpenAI && cfg.BaseURL == "" {
		missing = append(missing, "LLM_BASE_URL")
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("LLM_PROVIDER=%s requires %s", cfg.Provider, strings.Join(missing, ", "))
	}
	return cfg, nil
}

func New(cfg Config) (usecase.VulnerabilityExtractor, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		return anthropic.NewClient(anthropic.Config{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}), nil
	case ProviderOpenAI:
		return openai.NewClient(openai.Config{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model})
	default:
		return nil, fmt.Errorf("unknown LLM provider %q", cfg.Provider)
	}
}
