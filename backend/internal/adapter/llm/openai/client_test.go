package openai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jmpberlin/nightwatch/backend/internal/adapter/llm/shared"
	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

const (
	testKey   = "sk-test-secret-key"
	testModel = "z-ai/glm-5.3-flash"
)

var testArticle = domain.Article{ID: "article-1", ContentCleaned: "Command injection in lodash < 4.17.21 (CVE-2021-23337)."}

const vulnArgs = `{"vulnerabilities":[{"CVE":"CVE-2021-23337","Severity":"HIGH","Description":"Command injection in lodash template","AffectedTechnologies":[{"Name":"lodash","VersionRange":"<4.17.21","PURL":"pkg:npm/lodash"}]}]}`

func toolCallResponse(t *testing.T, args string) string {
	t.Helper()
	return mustJSON(t, map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion", "created": 1, "model": testModel,
		"choices": []any{map[string]any{
			"index": 0, "finish_reason": "tool_calls",
			"message": map[string]any{
				"role": "assistant", "content": nil, "reasoning_content": "thinking...",
				"tool_calls": []any{map[string]any{
					"id": "call_1", "type": "function",
					"function": map[string]any{"name": shared.ToolName, "arguments": args},
				}},
			},
		}},
	})
}

func contentResponse(t *testing.T, content, finishReason string) string {
	t.Helper()
	return mustJSON(t, map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion", "created": 1, "model": testModel,
		"choices": []any{map[string]any{
			"index": 0, "finish_reason": finishReason,
			"message": map[string]any{"role": "assistant", "content": content},
		}},
	})
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type step struct {
	status int
	body   string
}

func newSequencedServer(t *testing.T, steps ...step) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := min(int(calls.Add(1))-1, len(steps)-1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("retry-after-ms", "1")
		w.WriteHeader(steps[n].status)
		_, _ = io.WriteString(w, steps[n].body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(Config{BaseURL: baseURL + "/v1", APIKey: testKey, Model: testModel})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func extractOne(t *testing.T, c *Client) domain.ArticleExtractionResult {
	t.Helper()
	results := c.ExtractVulnerabilities([]domain.Article{testArticle})
	if len(results) != 1 || results[0].ArticleID != testArticle.ID {
		t.Fatalf("unexpected results: %+v", results)
	}
	return results[0]
}

func assertLodash(t *testing.T, vulns []domain.Vulnerability) {
	t.Helper()
	if len(vulns) != 1 {
		t.Fatalf("got %d vulnerabilities, want 1", len(vulns))
	}
	v := vulns[0]
	if v.CVE != "CVE-2021-23337" || v.Severity != domain.SeverityHigh {
		t.Errorf("unexpected vulnerability: %+v", v)
	}
	if len(v.SourceArticleIDs) != 1 || v.SourceArticleIDs[0] != testArticle.ID {
		t.Errorf("SourceArticleIDs = %v", v.SourceArticleIDs)
	}
	want := domain.AffectedTechnology{Name: "lodash", VersionRange: "<4.17.21", PURL: "pkg:npm/lodash"}
	if len(v.AffectedTechnologies) != 1 || v.AffectedTechnologies[0] != want {
		t.Errorf("AffectedTechnologies = %+v, want [%+v]", v.AffectedTechnologies, want)
	}
}

func TestExtract_ToolCall_RequestShape(t *testing.T) {
	var path, auth string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth = r.URL.Path, r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, toolCallResponse(t, vulnArgs))
	}))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	assertLodash(t, res.Vulnerabilities)

	if path != "/v1/chat/completions" {
		t.Errorf("path = %q", path)
	}
	if auth != "Bearer "+testKey {
		t.Errorf("Authorization = %q", auth)
	}
	if body["model"] != testModel || body["max_tokens"] != float64(maxTokensIncludingReasoning) {
		t.Errorf("model/max_tokens = %v / %v", body["model"], body["max_tokens"])
	}

	msgs, _ := body["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("messages = %v", body["messages"])
	}
	sys, user := msgs[0].(map[string]any), msgs[1].(map[string]any)
	if sys["role"] != "system" || sys["content"] != shared.Prompt {
		t.Errorf("system message = %v", sys)
	}
	if user["role"] != "user" || user["content"] != testArticle.ContentCleaned {
		t.Errorf("user message = %v", user)
	}

	tools, _ := body["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %v", body["tools"])
	}
	fn := tools[0].(map[string]any)["function"].(map[string]any)
	params, _ := fn["parameters"].(map[string]any)
	if fn["name"] != shared.ToolName || params["type"] != "object" || params["properties"] == nil {
		t.Errorf("tool function = %v", fn)
	}

	choice, _ := body["tool_choice"].(map[string]any)
	if choice["type"] != "function" || choice["function"].(map[string]any)["name"] != shared.ToolName {
		t.Errorf("tool_choice = %v", body["tool_choice"])
	}
}

func TestExtract_EmptyVulnerabilities(t *testing.T) {
	srv, _ := newSequencedServer(t, step{200, toolCallResponse(t, `{"vulnerabilities":[]}`)})
	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil || len(res.Vulnerabilities) != 0 {
		t.Errorf("got %+v, %v; want empty, nil", res.Vulnerabilities, res.Err)
	}
}

func TestExtract_ContentFallback(t *testing.T) {
	for name, content := range map[string]string{
		"plain JSON":  vulnArgs,
		"fenced JSON": "```json\n" + vulnArgs + "\n```",
	} {
		t.Run(name, func(t *testing.T) {
			srv, _ := newSequencedServer(t, step{200, contentResponse(t, content, "stop")})
			res := extractOne(t, newTestClient(t, srv.URL))
			if res.Err != nil {
				t.Fatalf("unexpected error: %v", res.Err)
			}
			assertLodash(t, res.Vulnerabilities)
		})
	}
}

func TestExtract_ParseErrors(t *testing.T) {
	cases := map[string]struct {
		body    string
		wantErr string
	}{
		"malformed tool arguments": {toolCallResponse(t, `{"vulnerabilities":[{"CVE":`), "tool call arguments"},
		"content is prose":         {contentResponse(t, "I found no vulnerabilities.", "stop"), "content fallback"},
		"content is other JSON":    {contentResponse(t, `{"answer":"none"}`, "stop"), "missing"},
		"empty content, truncated": {contentResponse(t, "", "length"), `finish_reason="length"`},
		"no choices":               {`{"id":"x","object":"chat.completion","choices":[]}`, "no choices"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv, _ := newSequencedServer(t, step{200, tc.body})
			res := extractOne(t, newTestClient(t, srv.URL))
			if res.Err == nil || !strings.Contains(res.Err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want containing %q", res.Err, tc.wantErr)
			}
			if res.Vulnerabilities != nil {
				t.Errorf("expected no vulnerabilities on error, got %+v", res.Vulnerabilities)
			}
		})
	}
}

func TestExtract_HTTPErrors(t *testing.T) {
	ok := step{200, toolCallResponse(t, vulnArgs)}
	cases := map[string]struct {
		steps     []step
		wantCalls int32
		wantErr   string // empty means success expected
	}{
		"401 is not retried": {
			steps:     []step{{401, `{"error":{"message":"invalid api key","type":"auth_error"}}`}},
			wantCalls: 1,
			wantErr:   "HTTP 401: invalid api key",
		},
		"500 is retried once, then succeeds": {
			steps:     []step{{500, `{"error":{"message":"boom"}}`}, ok},
			wantCalls: 2,
		},
		"429 is retried once, then succeeds": {
			steps:     []step{{429, `{"error":{"message":"slow down"}}`}, ok},
			wantCalls: 2,
		},
		"503 twice gives up after one retry": {
			steps:     []step{{503, `{"error":{"message":"unavailable"}}`}},
			wantCalls: 2,
			wantErr:   "HTTP 503: unavailable",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv, calls := newSequencedServer(t, tc.steps...)
			res := extractOne(t, newTestClient(t, srv.URL))
			if got := calls.Load(); got != tc.wantCalls {
				t.Errorf("calls = %d, want %d", got, tc.wantCalls)
			}
			if res.Err != nil && strings.Contains(res.Err.Error(), testKey) {
				t.Errorf("error leaks API key: %v", res.Err)
			}
			if tc.wantErr == "" {
				if res.Err != nil {
					t.Fatalf("unexpected error: %v", res.Err)
				}
				assertLodash(t, res.Vulnerabilities)
				return
			}
			if res.Err == nil || !strings.Contains(res.Err.Error(), tc.wantErr) {
				t.Errorf("error = %v, want containing %q", res.Err, tc.wantErr)
			}
		})
	}
}

func TestNewClient_Validation(t *testing.T) {
	for name, cfg := range map[string]Config{
		"missing base URL": {APIKey: "k", Model: "m"},
		"missing key":      {BaseURL: "http://x", Model: "m"},
		"missing model":    {BaseURL: "http://x", APIKey: "k"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewClient(cfg); err == nil {
				t.Error("expected error")
			}
		})
	}
}
