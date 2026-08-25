package quota

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQuotaAsyncFetchAndCache(t *testing.T) {
	called := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		json.NewEncoder(w).Encode([]wireWindow{
			{Label: "monthly", Pct: 25.0, ResetsIn: 1800},
		})
	}))
	defer ts.Close()

	c := New(ts.URL)

	// 1. First call with wait=false spawns background fetch
	c.Quota(false)
	time.Sleep(50 * time.Millisecond)

	// 2. Second call should return cached window
	cached := c.Quota(false)
	if len(cached) != 1 || cached[0].Label != "monthly" {
		t.Fatalf("expected cached monthly window, got %+v", cached)
	}

	// 3. Cache duration hit
	c.Quota(true)
	if len(c.cached) != 1 {
		t.Error("expected cache to remain populated")
	}
}

func TestQuotaErrorStatusesAndFormats(t *testing.T) {
	// 1. HTTP 500 error
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tsErr.Close()

	cErr := New(tsErr.URL)
	if _, err := cErr.Fetch(); err == nil {
		t.Error("expected Fetch to error on HTTP 500")
	}

	// 2. Sync wait on failed fetch returns nil
	if windows := cErr.Quota(true); windows != nil {
		t.Errorf("expected nil on failed sync quota fetch, got %+v", windows)
	}

	// 3. Unrecognised body format
	if _, err := parseWindows([]byte(`{"invalid":true}`)); err == nil {
		t.Error("expected parseWindows to error on unrecognised format")
	}

	// 4. Wrapper with "quota" key format
	quotaJSON := []byte(`{"quota":[{"label":"daily","pct":80.0,"resets_in":600}]}`)
	windows, err := parseWindows(quotaJSON)
	if err != nil || len(windows) != 1 || windows[0].Label != "daily" {
		t.Errorf("parseWindows wrapper quota failed: %v, windows=%+v", err, windows)
	}

	// 5. Single object format
	singleJSON := []byte(`{"label":"hourly","pct":12.5,"resets_in_s":120}`)
	windowsSingle, err := parseWindows(singleJSON)
	if err != nil || len(windowsSingle) != 1 || windowsSingle[0].Label != "hourly" {
		t.Errorf("parseWindows single failed: %v, windows=%+v", err, windowsSingle)
	}
}

func FuzzParseWindows(f *testing.F) {
	f.Add([]byte(`[{"label":"5h","pct":50.0,"resets_in":300}]`))
	f.Add([]byte(`{"windows":[{"label":"7d","pct":20.0,"resets_in_s":600}]}`))
	f.Add([]byte(`{"label":"single","pct":10.0}`))
	f.Add([]byte(`random corrupt payload`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseWindows(data)
	})
}
