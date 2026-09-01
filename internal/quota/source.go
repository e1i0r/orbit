package quota

// Where a window is read from, and the one protocol that answers today.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MaxKeyLen is the maximum length of an account key preserved for display.
const MaxKeyLen = 12

// fetchTimeout is how long one look at a source may take.
//
// It is a second because this is drawn on a status bar several times a
// second: a source that needs longer than that to answer is a source whose
// answer arrives after the frame it was for, and the backoff above is the
// better way to wait for it.
const fetchTimeout = time.Second

// Source is where one engine's quota windows come from.
//
// It is an interface because the answer differs per engine and the question
// does not. What every source has in common is the shape it answers in —
// Window, which is what a reader is shown — so an engine whose limits arrive
// some other way can be added behind this without the status bar, the
// budgets or the stats learning a second shape.
//
// One protocol implements it today: a proxy that answers GET /quota, pointed
// at whichever base URL an engine is driven through. The engines with no
// such proxy have no source at all, and that is an answer this package
// states rather than a hole it fills with zero.
type Source interface {
	Read() ([]Window, error)
}

// proxy is a base URL that answers GET /quota.
type proxy struct {
	baseURL string
	client  *http.Client
}

// Read asks the proxy once, and turns what came back into windows.
func (p proxy) Read() ([]Window, error) {
	req, err := http.NewRequest(http.MethodGet, p.baseURL+"/quota", nil)
	if err != nil {
		return nil, fmt.Errorf("build quota request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quota HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read quota body: %w", err)
	}

	return parseWindows(body)
}

// wireWindow is one window as a proxy writes it. The two spellings of the
// countdown are both in use by proxies in the wild.
type wireWindow struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Pct       float64 `json:"pct"`
	ResetsInS int64   `json:"resets_in_s"`
	ResetsIn  int64   `json:"resets_in"`
}

// wireHeadroom is the shape the proxy in front of Anthropic answers in: one
// object holding the two windows a subscription actually has, each with the
// share of it already used and the seconds until it comes back.
//
// It is named after the proxy rather than after a vendor because that is
// what it is a fact about — the endpoint's own spelling of the same two
// numbers every other shape here carries.
type wireHeadroom struct {
	Subscription struct {
		Latest struct {
			FiveHour headroomWindow `json:"five_hour"`
			SevenDay headroomWindow `json:"seven_day"`
			Prefix   string         `json:"token_prefix"`
		} `json:"latest"`
	} `json:"subscription_window"`
}

// headroomWindow is one of those two windows.
type headroomWindow struct {
	Pct  float64 `json:"utilization_pct"`
	Secs float64 `json:"seconds_to_reset"`
}

// windows is the pair as this package speaks of them, dropping a window the
// proxy answered nothing for: a plan with no seven-day limit reports the
// field as zeros, and a zeroed window drawn is 100% left of a window that
// does not exist.
func (h wireHeadroom) windows() []Window {
	latest := h.Subscription.Latest

	var out []Window

	for _, w := range []struct {
		label string
		win   headroomWindow
	}{{"5h", latest.FiveHour}, {"7d", latest.SevenDay}} {
		if w.win.Secs <= 0 && w.win.Pct <= 0 {
			continue
		}

		key := latest.Prefix
		if len(key) > MaxKeyLen {
			key = key[:MaxKeyLen]
		}

		out = append(out, Window{
			Key:      key,
			Label:    w.label,
			Pct:      w.win.Pct,
			ResetsIn: time.Duration(w.win.Secs) * time.Second,
		})
	}

	return out
}

// parseWindows reads the four shapes a proxy answers in: a bare array, an
// object wrapping one, a single window on its own, and the nested pair a
// headroom proxy reports a subscription as.
func parseWindows(body []byte) ([]Window, error) {
	var list []wireWindow
	if err := json.Unmarshal(body, &list); err == nil {
		return toWindows(list), nil
	}

	var wrapper struct {
		Windows []wireWindow `json:"windows"`
		Quota   []wireWindow `json:"quota"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		if len(wrapper.Windows) > 0 {
			return toWindows(wrapper.Windows), nil
		}

		if len(wrapper.Quota) > 0 {
			return toWindows(wrapper.Quota), nil
		}
	}

	var single wireWindow
	if err := json.Unmarshal(body, &single); err == nil && (single.Label != "" || single.Pct > 0) {
		return toWindows([]wireWindow{single}), nil
	}

	var nested wireHeadroom
	if err := json.Unmarshal(body, &nested); err == nil {
		if windows := nested.windows(); len(windows) > 0 {
			return windows, nil
		}
	}

	return nil, errors.New("unrecognised quota response format")
}

func toWindows(list []wireWindow) []Window {
	out := make([]Window, 0, len(list))
	for _, w := range list {
		key := w.Key
		if len(key) > MaxKeyLen {
			key = key[:MaxKeyLen]
		}

		res := w.ResetsInS
		if res == 0 {
			res = w.ResetsIn
		}

		out = append(out, Window{
			Key:      key,
			Label:    w.Label,
			Pct:      w.Pct,
			ResetsIn: time.Duration(res) * time.Second,
		})
	}

	return out
}

// newProxy is a Source for a base URL, or nothing when nobody named one.
func newProxy(baseURL string) Source {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil
	}

	return proxy{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: fetchTimeout},
	}
}
