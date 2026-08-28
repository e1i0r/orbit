package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
)

func TestSuperviseExecutesEngineAndRecordsAnswer(t *testing.T) {
	s, _ := fixture(t)
	fake := &engine.Fake{
		Output: "Reviewed all tasks, all systems nominal.",
	}

	res, err := Supervise(context.Background(), s, fake, "how is the board doing?")
	if err != nil {
		t.Fatalf("Supervise failed: %v", err)
	}

	if res != "Reviewed all tasks, all systems nominal." {
		t.Errorf("res = %q", res)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("fake.Calls len = %d, want 1", len(fake.Calls))
	}

	if !strings.Contains(fake.Calls[0].Prompt, "how is the board doing?") {
		t.Errorf("prompt missing question: %s", fake.Calls[0].Prompt)
	}

	// Verify recorded in supervisor thread
	events, err := SupervisorEvents(s)
	if err != nil {
		t.Fatalf("SupervisorEvents: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}

	if events[0].Text != "Reviewed all tasks, all systems nominal." || events[0].Data["by"] != "fake" {
		t.Errorf("event[0] = %+v", events[0])
	}
}

func TestSuperviseRefusesNilStoreOrEngineOrEmptyPrompt(t *testing.T) {
	s, _ := fixture(t)
	fake := &engine.Fake{Output: "ok"}

	if _, err := Supervise(context.Background(), nil, fake, "msg"); err == nil {
		t.Error("Supervise on nil store answered nil, want error")
	}

	if _, err := Supervise(context.Background(), s, nil, "msg"); err == nil {
		t.Error("Supervise on nil engine answered nil, want error")
	}

	if _, err := Supervise(context.Background(), s, fake, "   "); err == nil {
		t.Error("Supervise on empty prompt answered nil, want error")
	}
}

func TestAutoSuperviseBuildsPromptWithTaskIDs(t *testing.T) {
	s, _ := fixture(t)
	fake := &engine.Fake{
		Output: "Addressed failures in ORB-10 and ORB-12.",
	}

	res, err := AutoSupervise(context.Background(), s, fake, []string{"ORB-10", "ORB-12"})
	if err != nil {
		t.Fatalf("AutoSupervise failed: %v", err)
	}

	if !strings.Contains(fake.Calls[0].Prompt, "ORB-10, ORB-12") {
		t.Errorf("prompt missing task list: %s", fake.Calls[0].Prompt)
	}

	if res != "Addressed failures in ORB-10 and ORB-12." {
		t.Errorf("res = %q", res)
	}
}
