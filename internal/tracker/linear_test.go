package tracker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestFetchLinearReportsWhatTheAPIRefused is the fix.
//
// GraphQL answers 200 to a request it refused, with the reason in errors and
// data.issue null. Reading only data meant a revoked key and an issue on
// another workspace both returned ("", "", nil) — an empty issue and no
// error — and a task was created from it.
func TestFetchLinearReportsWhatTheAPIRefused(t *testing.T) {
	for _, c := range []struct {
		name, body, want string
	}{
		{
			"a revoked key",
			`{"errors":[{"message":"Authentication required, not authenticated"}],"data":null}`,
			"not authenticated",
		},
		{
			"an issue this key cannot see",
			`{"data":{"issue":null}}`,
			"ENG-1",
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if _, err := w.Write([]byte(c.body)); err != nil {
				t.Errorf("write body: %v", err)
			}
		}))

		old := linearEndpoint
		linearEndpoint = ts.URL

		title, desc, err := FetchLinear(context.Background(), "lin_api_key", "ENG-1")

		linearEndpoint = old

		ts.Close()

		if err == nil {
			t.Errorf("%s: FetchLinear returned title %q desc %q and no error", c.name, title, desc)
			continue
		}

		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error = %v, want it to mention %q", c.name, err, c.want)
		}
	}
}

// TestFetchLinearReturnsTheIssue keeps the happy path honest: the guards
// above must not turn a real answer into a refusal.
func TestFetchLinearReturnsTheIssue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`{"data":{"issue":{"title":"Fix the thing","description":"It is broken"}}}`)); err != nil {
			t.Errorf("write body: %v", err)
		}
	}))
	defer ts.Close()

	old := linearEndpoint
	linearEndpoint = ts.URL

	defer func() { linearEndpoint = old }()

	title, desc, err := FetchLinear(context.Background(), "lin_api_key", "ENG-1")
	if err != nil {
		t.Fatalf("FetchLinear: %v", err)
	}

	if title != "Fix the thing" || desc != "It is broken" {
		t.Errorf("got %q / %q, want the issue's title and description", title, desc)
	}
}
