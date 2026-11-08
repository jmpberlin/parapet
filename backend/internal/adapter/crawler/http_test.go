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
