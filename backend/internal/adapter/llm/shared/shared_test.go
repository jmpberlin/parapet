package shared

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmpberlin/nightwatch/backend/internal/domain"
)

var art = domain.Article{ID: "a1"}

func TestParsePayload(t *testing.T) {
	raw := `{"vulnerabilities":[{"CVE":"CVE-1","Severity":"CRITICAL","Description":"d","AffectedTechnologies":[{"Name":"log4j","VersionRange":"<2.17.0","PURL":"pkg:maven/org.apache.logging.log4j/log4j-core"}]}]}`
	vulns, err := ParsePayload([]byte(raw), art)
	if err != nil {
		t.Fatal(err)
	}
	if len(vulns) != 1 {
		t.Fatalf("got %d vulns", len(vulns))
	}
	v := vulns[0]
	if v.CVE != "CVE-1" || v.Severity != domain.SeverityCritical || v.Description != "d" {
		t.Errorf("unexpected %+v", v)
	}
	if v.SourceArticleIDs[0] != "a1" || v.CreatedAt.IsZero() {
		t.Errorf("attribution/timestamp missing: %+v", v)
	}
	want := domain.AffectedTechnology{Name: "log4j", VersionRange: "<2.17.0", PURL: "pkg:maven/org.apache.logging.log4j/log4j-core"}
	if v.AffectedTechnologies[0] != want {
		t.Errorf("tech = %+v", v.AffectedTechnologies[0])
	}
}

func TestParsePayload_LenientOnMissingKey(t *testing.T) {
	vulns, err := ParsePayload([]byte(`{}`), art)
	if err != nil || len(vulns) != 0 {
		t.Errorf("got %v, %v; want empty, nil", vulns, err)
	}
}

func TestParsePayload_Malformed(t *testing.T) {
	if _, err := ParsePayload([]byte(`{"vulnerabilities":`), art); err == nil {
		t.Error("expected error")
	}
}

func TestParsePayloadStrict(t *testing.T) {
	if _, err := ParsePayloadStrict([]byte(`{"other":1}`), art); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("err = %v, want missing key", err)
	}
	if _, err := ParsePayloadStrict([]byte(`[1,2]`), art); err == nil {
		t.Error("expected error for non-object")
	}
	vulns, err := ParsePayloadStrict([]byte(`{"vulnerabilities":[]}`), art)
	if err != nil || len(vulns) != 0 {
		t.Errorf("got %v, %v", vulns, err)
	}
}

// encoding/json matches keys case-insensitively, but "version_range" never matches "VersionRange".
func TestPromptMatchesSchemaFieldNames(t *testing.T) {
	for _, field := range []string{"Name", "PURL", "VersionRange"} {
		if !strings.Contains(Prompt, "- "+field+":") {
			t.Errorf("prompt does not describe field %q", field)
		}
	}
	if strings.Contains(Prompt, "version_range") {
		t.Error("prompt still references snake_case version_range")
	}
}

func TestSchemaIsFreshAndSerializable(t *testing.T) {
	a := Schema()
	a["type"] = "mutated"
	if Schema()["type"] != "object" {
		t.Error("Schema() returns shared state")
	}
	if _, err := json.Marshal(Schema()); err != nil {
		t.Fatal(err)
	}
}

func TestProcessConcurrently_BoundsConcurrency(t *testing.T) {
	articles := make([]domain.Article, 20)
	for i := range articles {
		articles[i] = domain.Article{ID: string(rune('A' + i))}
	}

	var inFlight, peak atomic.Int32
	var mu sync.Mutex
	results := ProcessConcurrently(articles, func(a domain.Article) domain.ArticleExtractionResult {
		n := inFlight.Add(1)
		mu.Lock()
		if n > peak.Load() {
			peak.Store(n)
		}
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		inFlight.Add(-1)
		return domain.ArticleExtractionResult{ArticleID: a.ID}
	})

	if len(results) != len(articles) {
		t.Fatalf("got %d results", len(results))
	}
	if p := peak.Load(); p > MaxConcurrency {
		t.Errorf("peak concurrency %d exceeds %d", p, MaxConcurrency)
	}
}

func TestProcessConcurrently_Empty(t *testing.T) {
	results := ProcessConcurrently(nil, func(domain.Article) domain.ArticleExtractionResult {
		t.Fatal("should not be called")
		return domain.ArticleExtractionResult{}
	})
	if results == nil || len(results) != 0 {
		t.Errorf("got %v, want empty non-nil slice", results)
	}
}
