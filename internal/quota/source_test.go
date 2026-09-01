package quota

import (
	"testing"
	"time"
)

// A headroom proxy answers with the two windows a subscription has nested
// inside one object, which is neither an array nor a window: the shape this
// reads is the one the proxy on a real machine actually returns.
func TestTheNestedSubscriptionShapeIsReadAsTwoWindows(t *testing.T) {
	body := []byte(`{"subscription_window":{"latest":{
		"five_hour":{"utilization_pct":1.0,"seconds_to_reset":17368.79},
		"seven_day":{"utilization_pct":76.0,"seconds_to_reset":188968.79},
		"extra_usage":{"is_enabled":false,"utilization_pct":0.0},
		"token_prefix":"sk-ant-oat01-longer-than-the-cap"}}}`)

	windows, err := parseWindows(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(windows) != 2 {
		t.Fatalf("windows: got %d, want 2 — extra_usage is not a window", len(windows))
	}

	if windows[0].Label != "5h" || windows[0].Pct != 1 {
		t.Errorf("first window: got %+v, want 1%% of 5h", windows[0])
	}

	if want := 17368 * time.Second; windows[0].ResetsIn != want {
		t.Errorf("resets in: got %v, want %v", windows[0].ResetsIn, want)
	}

	if windows[1].Label != "7d" || windows[1].Pct != 76 {
		t.Errorf("second window: got %+v, want 76%% of 7d", windows[1])
	}

	if len(windows[0].Key) > MaxKeyLen {
		t.Errorf("key: %q is longer than the cap", windows[0].Key)
	}
}

// A plan with no seven-day limit reports it as zeros, and a window nobody
// has is not a window that is empty.
func TestAWindowTheProxyAnsweredNothingForIsNotDrawn(t *testing.T) {
	body := []byte(`{"subscription_window":{"latest":{"five_hour":{"utilization_pct":42.0,"seconds_to_reset":600}}}}`)

	windows, err := parseWindows(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(windows) != 1 || windows[0].Label != "5h" {
		t.Fatalf("windows: got %+v, want the five-hour one alone", windows)
	}
}
