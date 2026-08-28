package tracker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLinear(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"data": map[string]any{
				"issue": map[string]any{
					"title":       "Fix tax calculation",
					"description": "Ensure VAT is calculated properly.",
				},
			},
		})
	}))
	defer ts.Close()

	// Empty apiKey or id
	if _, _, err := FetchLinear(context.Background(), "", "ID"); err == nil {
		t.Error("expected error with empty apiKey")
	}

	if _, _, err := FetchLinear(context.Background(), "key", ""); err == nil {
		t.Error("expected error with empty id")
	}
}
