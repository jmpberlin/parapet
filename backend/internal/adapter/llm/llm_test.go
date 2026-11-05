package llm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/anthropic"
	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/openai"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestConfigFromEnv(t *testing.T) {
	cases := map[string]struct {
		env     map[string]string
		want    Config
		wantErr string
	}{
		"anthropic": {
			env:  map[string]string{"LLM_PROVIDER": "anthropic", "LLM_API_KEY": "k", "LLM_MODEL": "claude-haiku-4-5"},
			want: Config{Provider: "anthropic", APIKey: "k", Model: "claude-haiku-4-5"},
		},
		"anthropic with custom base URL": {
			env:  map[string]string{"LLM_PROVIDER": "anthropic", "LLM_API_KEY": "k", "LLM_MODEL": "m", "LLM_BASE_URL": "https://proxy.example/v1"},
			want: Config{Provider: "anthropic", APIKey: "k", Model: "m", BaseURL: "https://proxy.example/v1"},
		},
		"openai (TensorX)": {
			env: map[string]string{
				"LLM_PROVIDER": "openai", "LLM_API_KEY": "tx-key",
				"LLM_BASE_URL": "https://api.tensorx.ai/v1", "LLM_MODEL": "z-ai/glm-5.3-flash",
			},
			want: Config{Provider: "openai", APIKey: "tx-key", BaseURL: "https://api.tensorx.ai/v1", Model: "z-ai/glm-5.3-flash"},
		},
		"provider is case-insensitive and trimmed": {
			env:  map[string]string{"LLM_PROVIDER": " OpenAI ", "LLM_API_KEY": "k", "LLM_MODEL": "m", "LLM_BASE_URL": "u"},
			want: Config{Provider: "openai", APIKey: "k", Model: "m", BaseURL: "u"},
		},
		"provider missing": {
			env:     map[string]string{"LLM_API_KEY": "k", "LLM_MODEL": "m"},
			wantErr: `LLM_PROVIDER must be "anthropic" or "openai", got ""`,
		},
		"unknown provider": {
			env:     map[string]string{"LLM_PROVIDER": "tensorx", "LLM_API_KEY": "k", "LLM_MODEL": "m"},
			wantErr: `got "tensorx"`,
		},
		"anthropic missing key and model": {
			env:     map[string]string{"LLM_PROVIDER": "anthropic"},
			wantErr: "LLM_PROVIDER=anthropic requires LLM_API_KEY, LLM_MODEL",
		},
		"openai missing everything": {
			env:     map[string]string{"LLM_PROVIDER": "openai"},
			wantErr: "LLM_PROVIDER=openai requires LLM_API_KEY, LLM_MODEL, LLM_BASE_URL",
		},
		"legacy CLAUDE_API_KEY is not used": {
			env:     map[string]string{"LLM_PROVIDER": "anthropic", "CLAUDE_API_KEY": "k", "LLM_MODEL": "m"},
			wantErr: "requires LLM_API_KEY",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ConfigFromEnv(envFrom(tc.env))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestNew_SelectsAdapter(t *testing.T) {
	anth, err := New(Config{Provider: "anthropic", APIKey: "k", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := anth.(*anthropic.Client); !ok {
		t.Errorf("anthropic provider built %T", anth)
	}

	oa, err := New(Config{Provider: "openai", APIKey: "k", BaseURL: "https://api.tensorx.ai/v1", Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := oa.(*openai.Client); !ok {
		t.Errorf("openai provider built %T", oa)
	}

	if _, err := New(Config{Provider: "nope"}); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestLogFieldsNeverContainKey(t *testing.T) {
	cfg := Config{Provider: "openai", APIKey: "super-secret", BaseURL: "u", Model: "m"}
	if s := fmt.Sprint(cfg.LogFields()...); strings.Contains(s, "super-secret") {
		t.Errorf("log fields leak key: %s", s)
	}
}
