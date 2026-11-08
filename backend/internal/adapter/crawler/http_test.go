package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRaw_RejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("a", maxResponseBodyBytes+1))
	}))
	defer srv.Close()

	_, err := fetchRaw(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestFetchRaw_AcceptsBodyAtExactLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("a", maxResponseBodyBytes))
	}))
	defer srv.Close()

	body, err := fetchRaw(srv.URL)
	require.NoError(t, err)
	assert.Len(t, body, maxResponseBodyBytes)
}

func TestFetchRaw_ReturnsResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<p>hello</p>")
	}))
	defer srv.Close()

	body, err := fetchRaw(srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "<p>hello</p>", string(body))
}

func TestFetchDocument_ParsesHTMLIntoDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!DOCTYPE html><html><body><h1>Title</h1></body></html>")
	}))
	defer srv.Close()

	doc, err := fetchDocument(srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "Title", doc.Find("h1").Text())
}

func TestFetchDocument_Non200StatusErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchDocument(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
}

func TestFetchDocument_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	_, err := fetchDocument(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch")
}
