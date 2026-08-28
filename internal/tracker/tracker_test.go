package tracker

import (
	"strings"
	"sync"
	"testing"
)

func TestParseAllProviders(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantKind    string
		wantID      string
		wantTitle   string
		wantPromptM string
	}{
		{
			name:        "linear issue with slug",
			url:         "https://linear.app/myorg/issue/PAY-105/retry-payment-webhooks-on-5xx",
			wantKind:    "linear",
			wantID:      "PAY-105",
			wantTitle:   "Retry payment webhooks on 5xx",
			wantPromptM: "Linear MCP tools",
		},
		{
			name:        "jira issue cloud",
			url:         "https://acmecorp.atlassian.net/browse/PROJ-456",
			wantKind:    "jira",
			wantID:      "PROJ-456",
			wantPromptM: "Jira MCP tools",
		},
		{
			name:        "github issue",
			url:         "https://github.com/e1i0r/orbit/issues/42",
			wantKind:    "github",
			wantID:      "GH-42",
			wantPromptM: "GitHub MCP tools",
		},
		{
			name:        "gitlab issue",
			url:         "https://gitlab.com/gitlab-org/gitlab/-/issues/12345",
			wantKind:    "gitlab",
			wantID:      "GL-12345",
			wantPromptM: "GitLab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iss, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.url, err)
			}

			if iss.Kind != tt.wantKind {
				t.Errorf("Kind = %q, want %q", iss.Kind, tt.wantKind)
			}

			if iss.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", iss.ID, tt.wantID)
			}

			if tt.wantTitle != "" && iss.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", iss.Title, tt.wantTitle)
			}

			prompt := FormatPrompt(iss)
			if !strings.Contains(prompt, tt.wantPromptM) {
				t.Errorf("FormatPrompt(%v) = %q, want to contain %q", iss, prompt, tt.wantPromptM)
			}
		})
	}
}

func TestIsTrackerURL(t *testing.T) {
	if !IsTrackerURL("https://linear.app/org/issue/ENG-1/foo") {
		t.Error("expected linear URL to match")
	}

	if !IsTrackerURL("https://jira.corp.com/browse/ENG-1") {
		t.Error("expected jira URL to match")
	}

	if IsTrackerURL("https://example.com/not/an/issue") {
		t.Error("expected non-tracker URL to not match")
	}
}

// TestRegisterIsSafeUnderTheRaceDetector. Register appended to a
// package-level slice while every reader ranged over the same one, with
// nothing serialising them.
func TestRegisterIsSafeUnderTheRaceDetector(t *testing.T) {
	var wg sync.WaitGroup

	for range 8 {
		wg.Add(2)

		go func() {
			defer wg.Done()

			Register(LinearProvider{})
		}()

		go func() {
			defer wg.Done()

			IsTrackerURL("https://linear.app/acme/issue/ENG-1")
		}()
	}

	wg.Wait()
}
