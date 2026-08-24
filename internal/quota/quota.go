package quota

// Package quota queries the Anthropic / proxy quota endpoint (GET /quota)
// to discover remaining limits and reset intervals.
//
// Absence of the proxy is not a failure: when ANTHROPIC_BASE_URL is unset,
// or unreachable, no quota is reported and no error is raised.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// CacheDuration is how long a good quota read is cached.
const CacheDuration = 30 * time.Second

// MaxKeyLen is the maximum length of an account key preserved for display.
const MaxKeyLen = 12

// Window is one quota window: percentage and countdown clock.
type Window struct {
	Key      string
	Label    string
	Pct      float64
	ResetsIn time.Duration
}

// Client fetches and caches quota windows from the proxy.
type Client struct {
	baseURL string
	client  *http.Client

	mu        sync.Mutex
	cached    []Window
	fetchedAt time.Time
	fetching  bool
}

// New builds a Client for the given base URL. An empty base URL yields nil.
func New(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 1 * time.Second},
	}
}

// FromEnv builds a Client from ANTHROPIC_BASE_URL.
func FromEnv() *Client {
	return New(os.Getenv("ANTHROPIC_BASE_URL"))
}

// Fetch requests GET /quota from the base URL synchronously.
func (c *Client) Fetch() ([]Window, error) {
	if c == nil || c.baseURL == "" {
		return nil, nil
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/quota", nil)
	if err != nil {
		return nil, fmt.Errorf("build quota request: %w", err)
	}
	resp, err := c.client.Do(req)
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

// Quota returns the cached quota windows immediately. If the cache is cold,
// a background goroutine is spawned to refresh it unless wait is true (used
// for single-frame CLI commands like --once).
func (c *Client) Quota(wait bool) []Window {
	if c == nil || c.baseURL == "" {
		return nil
	}
	c.mu.Lock()
	if time.Since(c.fetchedAt) < CacheDuration && c.cached != nil {
		cached := c.cached
		c.mu.Unlock()
		return cached
	}
	if wait {
		c.mu.Unlock()
		windows, err := c.Fetch()
		if err == nil && windows != nil {
			c.mu.Lock()
			c.cached = windows
			c.fetchedAt = time.Now()
			c.mu.Unlock()
			return windows
		}
		return nil
	}
	if !c.fetching {
		c.fetching = true
		c.mu.Unlock()
		go func() {
			windows, err := c.Fetch()
			c.mu.Lock()
			defer c.mu.Unlock()
			c.fetching = false
			if err == nil && windows != nil {
				c.cached = windows
				c.fetchedAt = time.Now()
			}
		}()
		c.mu.Lock()
	}
	cached := c.cached
	c.mu.Unlock()
	return cached
}

type wireWindow struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Pct       float64 `json:"pct"`
	ResetsInS int64   `json:"resets_in_s"`
	ResetsIn  int64   `json:"resets_in"`
}

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
