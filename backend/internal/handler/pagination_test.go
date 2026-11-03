package handler

import (
	"net/http/httptest"
	"testing"
)

func TestGetPaginationParams(t *testing.T) {
	cases := []struct {
		name        string
		query       string
		defaultVal  int
		wantPage    int
		wantLimit   int
		wantInvalid bool
	}{
		{name: "defaults with custom defaultLimit", query: "", defaultVal: 25, wantPage: 1, wantLimit: 25},
		{name: "provided values override defaults", query: "page=3&limit=5", defaultVal: 25, wantPage: 3, wantLimit: 5},
		{name: "page boundary maximum", query: "page=1000000", defaultVal: 10, wantPage: 1000000, wantLimit: 10},
		{name: "page boundary exceeded", query: "page=1000001", defaultVal: 10, wantInvalid: true},
		{name: "page zero", query: "page=0", defaultVal: 10, wantInvalid: true},
		{name: "page negative", query: "page=-2", defaultVal: 10, wantInvalid: true},
		{name: "page not a number", query: "page=abc", defaultVal: 10, wantInvalid: true},
		{name: "limit boundary maximum", query: "limit=100", defaultVal: 10, wantPage: 1, wantLimit: 100},
		{name: "limit boundary exceeded", query: "limit=101", defaultVal: 10, wantInvalid: true},
		{name: "limit zero", query: "limit=0", defaultVal: 10, wantInvalid: true},
		{name: "limit not a number", query: "limit=xyz", defaultVal: 10, wantInvalid: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tc.query, nil)

			page, limit, err := getPaginationParams(req, tc.defaultVal)

			if tc.wantInvalid {
				if err == nil {
					t.Fatalf("expected error for query %q", tc.query)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for query %q: %s", tc.query, err)
			}
			if page != tc.wantPage || limit != tc.wantLimit {
				t.Errorf("got page=%d limit=%d, want page=%d limit=%d", page, limit, tc.wantPage, tc.wantLimit)
			}
		})
	}
}

func TestCalculateTotalPages(t *testing.T) {
	cases := []struct {
		total, limit, want int
	}{
		{total: 0, limit: 10, want: 1},
		{total: 1, limit: 10, want: 1},
		{total: 10, limit: 10, want: 1},
		{total: 11, limit: 10, want: 2},
		{total: 20, limit: 10, want: 2},
		{total: 21, limit: 10, want: 3},
		{total: 100, limit: 100, want: 1},
		{total: 101, limit: 100, want: 2},
	}

	for _, tc := range cases {
		if got := calculateTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("calculateTotalPages(%d, %d) = %d, want %d", tc.total, tc.limit, got, tc.want)
		}
	}
}
