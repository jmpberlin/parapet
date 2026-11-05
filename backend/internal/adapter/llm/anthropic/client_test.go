package anthropic

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
	"github.com/jmpberlin/nightwatch/backend/internal/usecase"
)

const (
	testKey   = "ak-test-secret-key"
	testModel = "claude-haiku-4-5"
)

var testArticle = domain.Article{ID: "article-1", ContentCleaned: "Command injection in lodash < 4.17.21 (CVE-2021-23337)."}

const vulnPayload = `{"vulnerabilities":[{"CVE":"CVE-2021-23337","Severity":"HIGH","Description":"Command injection in lodash template","AffectedTechnologies":[{"Name":"lodash","VersionRange":"<4.17.21","PURL":"pkg:npm/lodash"}]}]}`

// Compile-time check that the adapter satisfies the contract main.go wires into the extract use case.
var _ usecase.VulnerabilityExtractor = (*Client)(nil)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func messageResponse(t *testing.T, blocks ...map[string]any) string {
	t.Helper()
	return mustJSON(t, map[string]any{
		"id":      "msg_1",
		"type":    "message",
		"role":    "assistant",
		"model":   testModel,
		"content": blocks,
		"usage":   map[string]any{"input_tokens": 10, "output_tokens": 20},
	})
}

func toolUseBlock(input string) map[string]any {
	return map[string]any{"type": "tool_use", "id": "toolu_1", "name": shared.ToolName, "input": json.RawMessage(input)}
}

func textBlock(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(Config{BaseURL: baseURL, APIKey: testKey, Model: testModel})
}

func jsonHandler(t *testing.T, body string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}
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
	if v.CVE != "CVE-2021-23337" || v.Severity != domain.SeverityHigh || v.Description != "Command injection in lodash template" {
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

func TestExtract_RequestShape(t *testing.T) {
	var path, apiKey, apiVersion string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		apiKey = r.Header.Get("X-Api-Key")
		apiVersion = r.Header.Get("anthropic-version")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, messageResponse(t, toolUseBlock(vulnPayload)))
	}))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	assertLodash(t, res.Vulnerabilities)

	if path != "/v1/messages" {
		t.Errorf("path = %q, want /v1/messages", path)
	}
	if apiKey != testKey {
		t.Errorf("X-Api-Key = %q", apiKey)
	}
	if apiVersion == "" {
		t.Error("anthropic-version header missing")
	}
	if body["model"] != testModel || body["max_tokens"] != float64(maxTokens) {
		t.Errorf("model/max_tokens = %v / %v", body["model"], body["max_tokens"])
	}

	sys, ok := body["system"].([]any)
	if !ok || len(sys) != 1 {
		t.Fatalf("system = %v", body["system"])
	}
	if sys[0].(map[string]any)["text"] != shared.Prompt {
		t.Errorf("system prompt = %v", sys[0])
	}

	msgs, ok := body["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %v", body["messages"])
	}
	msg := msgs[0].(map[string]any)
	if msg["role"] != "user" {
		t.Errorf("message role = %v", msg["role"])
	}
	content, ok := msg["content"].([]any)
	if !ok || len(content) != 1 || content[0].(map[string]any)["text"] != testArticle.ContentCleaned {
		t.Errorf("message content = %v", msg["content"])
	}

	tools, ok := body["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %v", body["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != shared.ToolName {
		t.Errorf("tool name = %v", tool["name"])
	}
	schema, ok := tool["input_schema"].(map[string]any)
	if !ok || schema["type"] != "object" || schema["properties"] == nil {
		t.Errorf("tool input_schema = %v", tool["input_schema"])
	}
	props, _ := schema["properties"].(map[string]any)
	vulnsProp, _ := props["vulnerabilities"].(map[string]any)
	if vulnsProp == nil || vulnsProp["type"] != "array" {
		t.Errorf("schema vulnerabilities property = %v", vulnsProp)
	}

	choice, ok := body["tool_choice"].(map[string]any)
	if !ok || choice["type"] != "tool" || choice["name"] != shared.ToolName {
		t.Errorf("tool_choice = %v", body["tool_choice"])
	}
}

func TestExtract_IgnoresTextBlocksAndParsesToolUse(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(t, messageResponse(t, textBlock("Analyzing..."), toolUseBlock(vulnPayload))))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	assertLodash(t, res.Vulnerabilities)
}

func TestExtract_EmptyPayload(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(t, messageResponse(t, toolUseBlock(`{"vulnerabilities":[]}`))))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil || len(res.Vulnerabilities) != 0 {
		t.Errorf("got %+v, %v; want empty, nil", res.Vulnerabilities, res.Err)
	}
}

func TestExtract_ConcatenatesMultipleToolUseBlocks(t *testing.T) {
	second := `{"vulnerabilities":[{"CVE":"CVE-2024-1234","Severity":"CRITICAL","Description":"another","AffectedTechnologies":[]}]}`
	srv := httptest.NewServer(jsonHandler(t, messageResponse(t, toolUseBlock(vulnPayload), toolUseBlock(second))))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if len(res.Vulnerabilities) != 2 {
		t.Fatalf("got %d vulnerabilities, want 2", len(res.Vulnerabilities))
	}
	if res.Vulnerabilities[0].CVE != "CVE-2021-23337" || res.Vulnerabilities[1].CVE != "CVE-2024-1234" {
		t.Errorf("order not preserved: %+v", res.Vulnerabilities)
	}
}

func TestExtract_NoToolUse(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(t, messageResponse(t, textBlock("no vulnerabilities found"))))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if len(res.Vulnerabilities) != 0 {
		t.Errorf("got %+v, want empty", res.Vulnerabilities)
	}
}

func TestExtract_InvalidToolInput(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(t, messageResponse(t, map[string]any{"type": "tool_use", "id": "toolu_1", "name": shared.ToolName, "input": "not-json"})))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err == nil || !strings.Contains(res.Err.Error(), "decode extraction payload") {
		t.Errorf("error = %v, want decode failure", res.Err)
	}
	if res.Vulnerabilities != nil {
		t.Errorf("expected no vulnerabilities on error, got %+v", res.Vulnerabilities)
	}
}

func TestExtract_MalformedResponseBody(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(t, "not json"))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err == nil {
		t.Error("expected error for malformed response body")
	}
	if res.Vulnerabilities != nil {
		t.Errorf("expected no vulnerabilities on error, got %+v", res.Vulnerabilities)
	}
}

func TestExtract_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	}))
	defer srv.Close()

	res := extractOne(t, newTestClient(t, srv.URL))
	if res.Err == nil {
		t.Fatal("expected error for HTTP 401")
	}
	if strings.Contains(res.Err.Error(), testKey) {
		t.Errorf("error leaks API key: %v", res.Err)
	}
	if res.Vulnerabilities != nil {
		t.Errorf("expected no vulnerabilities on error, got %+v", res.Vulnerabilities)
	}
}

func TestExtractVulnerabilities_MapsResultsToArticles(t *testing.T) {
	articles := []domain.Article{
		{ID: "a", ContentCleaned: "CVE-AAAA"},
		{ID: "b", ContentCleaned: "CVE-BBBB"},
		{ID: "c", ContentCleaned: "CVE-CCCC"},
		{ID: "d", ContentCleaned: "CVE-DDDD"},
		{ID: "e", ContentCleaned: "CVE-EEEE"},
		{ID: "f", ContentCleaned: "CVE-FFFF"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		cve := req.Messages[0].Content[0].Text
		payload := mustJSON(t, map[string]any{
			"vulnerabilities": []any{map[string]any{
				"CVE": cve, "Severity": "LOW", "Description": "", "AffectedTechnologies": []any{},
			}},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, messageResponse(t, toolUseBlock(payload)))
	}))
	defer srv.Close()

	results := newTestClient(t, srv.URL).ExtractVulnerabilities(articles)
	if len(results) != len(articles) {
		t.Fatalf("got %d results, want %d", len(results), len(articles))
	}
	byArticle := make(map[string]domain.ArticleExtractionResult, len(results))
	for _, r := range results {
		if _, dup := byArticle[r.ArticleID]; dup {
			t.Fatalf("duplicate result for article %q", r.ArticleID)
		}
		byArticle[r.ArticleID] = r
	}
	for _, a := range articles {
		r, ok := byArticle[a.ID]
		if !ok {
			t.Fatalf("no result for article %q", a.ID)
		}
		if r.Err != nil {
			t.Errorf("article %s: %v", a.ID, r.Err)
		}
		if len(r.Vulnerabilities) != 1 || r.Vulnerabilities[0].CVE != a.ContentCleaned {
			t.Errorf("article %s: got %+v", a.ID, r.Vulnerabilities)
		}
	}
}

func TestExtractVulnerabilities_EmptyInput(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, messageResponse(t, toolUseBlock(vulnPayload)))
	}))
	defer srv.Close()

	results := newTestClient(t, srv.URL).ExtractVulnerabilities(nil)
	if results == nil || len(results) != 0 {
		t.Errorf("got %v, want empty non-nil slice", results)
	}
	if calls.Load() != 0 {
		t.Errorf("API called %d times for empty input", calls.Load())
	}
}

func TestNewClient_BaseURLUsedWhenSet(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(t, messageResponse(t, toolUseBlock(vulnPayload))))
	defer srv.Close()

	res := extractOne(t, NewClient(Config{BaseURL: srv.URL, APIKey: testKey, Model: testModel}))
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	assertLodash(t, res.Vulnerabilities)
}
