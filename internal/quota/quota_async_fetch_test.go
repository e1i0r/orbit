package quota

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestQuotaAsyncFetchAndCache(t *testing.T) {
	called := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		_ = json.NewEncoder(w).Encode([]wireWindow{ //nolint:errcheck // test HTTP handler response
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
		_, _ = parseWindows(data) //nolint:errcheck // fuzz testing against arbitrary payloads
	})
}

// TestAProxyThatIsDownIsNotAskedOnEveryFrame is the regression. The status
// bar calls Quota on every frame it draws, and a failure used to leave
// nothing behind: the cache stayed cold, so the next frame started another
// request, and a proxy that was down turned the corner of the status bar into
// a connection attempt several times a second for as long as the window
// stayed open.
func TestAProxyThatIsDownIsNotAskedOnEveryFrame(t *testing.T) {
	var calls atomic.Int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := New(ts.URL)
	if got := c.Quota(true); got != nil {
		t.Fatalf("Quota = %+v, want nothing from a proxy answering 500", got)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("the first read asked %d times, want 1", got)
	}

	// Two seconds of frames, at the rate the spinner draws them.
	for range 50 {
		c.Quota(false)
	}

	time.Sleep(50 * time.Millisecond) // let any fetch that did go out land

	if got := calls.Load(); got != 1 {
		t.Errorf("the proxy was asked %d times over 50 frames, want the one read that failed", got)
	}

	if c.backoff != firstBackoff {
		t.Errorf("backoff = %s, want the first step of %s", c.backoff, firstBackoff)
	}
}

// TestTheBackoffGrowsAndThenGivesWayToAGoodRead. Reaching in to expire the
// hold-off is what a test can do instead of waiting five seconds for it, and
// the field is the whole mechanism: nothing else decides when the next
// request goes out.
func TestTheBackoffGrowsAndThenGivesWayToAGoodRead(t *testing.T) {
	var up atomic.Bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !up.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		_ = json.NewEncoder(w).Encode([]wireWindow{ //nolint:errcheck // test HTTP handler response
			{Label: "weekly", Pct: 40.0, ResetsIn: 3600},
		})
	}))
	defer ts.Close()

	c := New(ts.URL)

	want := []time.Duration{firstBackoff, 2 * firstBackoff, 4 * firstBackoff}
	for i, step := range want {
		c.Quota(true)

		if c.backoff != step {
			t.Fatalf("failure %d left a backoff of %s, want %s", i+1, c.backoff, step)
		}

		c.nextTry = time.Now() // the hold-off, expired
	}

	up.Store(true)

	windows := c.Quota(true)
	if len(windows) != 1 || windows[0].Label != "weekly" {
		t.Fatalf("Quota = %+v, want the window the proxy is now serving", windows)
	}

	if c.backoff != 0 {
		t.Errorf("backoff = %s after a good read, want it back to nothing", c.backoff)
	}

	if d := time.Until(c.nextTry); d < CacheDuration-time.Second {
		t.Errorf("the next read is due in %s, want it held for the cache duration of %s", d, CacheDuration)
	}
}
