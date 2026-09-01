package quota

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAClientWithNoBaseURLIsNoClient(t *testing.T) {
	c := New("  ")
	if c != nil {
		t.Fatalf("want nil client when nobody named a base URL, got %+v", c)
	}

	if got := c.Quota(true); got != nil {
		t.Fatalf("want nil quota from nil client, got %+v", got)
	}

	windows, err := c.Fetch()
	if windows != nil || err != nil {
		t.Fatalf("a nil client fetched %+v, %v; want nothing and no failure", windows, err)
	}
}

func TestQuotaFetchArray(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quota" {
			http.NotFound(w, r)
			return
		}

		_ = json.NewEncoder(w).Encode([]wireWindow{ //nolint:errcheck // test HTTP handler response
			{
				Key:       "sk-ant-1234567890extra",
				Label:     "5h",
				Pct:       62.0,
				ResetsInS: 8040,
			},
		})
	}))
	defer ts.Close()

	c := New(ts.URL)

	windows, err := c.Fetch()
	if err != nil {
		t.Fatalf("fetch quota: %v", err)
	}

	if len(windows) != 1 {
		t.Fatalf("want 1 window, got %d", len(windows))
	}

	w := windows[0]
	if len(w.Key) > MaxKeyLen {
		t.Errorf("key %q exceeds MaxKeyLen %d", w.Key, MaxKeyLen)
	}

	if w.Key != "sk-ant-12345" {
		t.Errorf("key = %q, want %q", w.Key, "sk-ant-12345")
	}

	if w.Label != "5h" {
		t.Errorf("label = %q, want %q", w.Label, "5h")
	}

	if w.Pct != 62.0 {
		t.Errorf("pct = %v, want 62.0", w.Pct)
	}

	if w.ResetsIn != 8040*time.Second {
		t.Errorf("resetsIn = %v, want %v", w.ResetsIn, 8040*time.Second)
	}
}

func TestQuotaFetchWrapped(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"windows":[{"label":"7d","pct":45.5,"resets_in":3600}]}`)) //nolint:errcheck // test HTTP handler response
	}))
	defer ts.Close()

	c := New(ts.URL)

	windows, err := c.Fetch()
	if err != nil {
		t.Fatalf("fetch quota: %v", err)
	}

	if len(windows) != 1 {
		t.Fatalf("want 1 window, got %d", len(windows))
	}

	if windows[0].Label != "7d" || windows[0].Pct != 45.5 || windows[0].ResetsIn != 3600*time.Second {
		t.Errorf("unexpected window: %+v", windows[0])
	}
}

func TestQuotaSyncWait(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]wireWindow{ //nolint:errcheck // test HTTP handler response
			{Label: "day", Pct: 10.0, ResetsInS: 60},
		})
	}))
	defer ts.Close()

	c := New(ts.URL)

	windows := c.Quota(true)
	if len(windows) != 1 {
		t.Fatalf("want 1 window on sync wait, got %d", len(windows))
	}

	if windows[0].Label != "day" {
		t.Errorf("label = %q, want day", windows[0].Label)
	}
}

func TestQuotaErrorHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	c := New(ts.URL)
	if _, err := c.Fetch(); err == nil {
		t.Error("want error on 503, got nil")
	}

	if got := c.Quota(true); got != nil {
		t.Errorf("want nil quota on error, got %+v", got)
	}
}
