package engine

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// The bounded buffer is tested directly rather than through Claude.Run,
// because Run spawns the claude binary and a suite that spawns it is a suite
// that spends money. The rule for this package is the one claudeArgs and
// ParseStream already follow: everything worth asserting is a function over
// values, and the binary is never on the other end of it.

// fill returns n bytes of recognisable filler, so a test can say which end
// of a stream survived.
func fill(n int, c byte) []byte {
	return bytes.Repeat([]byte{c}, n)
}

func TestABoundedBufferKeepsAStreamThatFitsAndDropsNothing(t *testing.T) {
	b := &boundedBuffer{max: 64}
	if _, err := b.Write([]byte("a phase that said very little")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := b.String(); got != "a phase that said very little" {
		t.Errorf("the buffer holds %q, which is not what was written to it", got)
	}

	if b.dropped != 0 {
		t.Errorf("a stream under the cap dropped %d bytes", b.dropped)
	}
}

// TestABoundedBufferKeepsTheEndOfTheStream is the choice this type makes and
// the one thing a reader has to know about it. The answer is the last line
// of claude's stream, so the end is the half worth keeping.
func TestABoundedBufferKeepsTheEndOfTheStream(t *testing.T) {
	b := &boundedBuffer{max: 10}
	if _, err := b.Write([]byte("0123456789")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := b.Write([]byte("abcde")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := b.String(); got != "56789abcde" {
		t.Errorf("the buffer holds %q, want the last ten bytes written", got)
	}

	if b.dropped != 5 {
		t.Errorf("the buffer dropped %d bytes, want 5", b.dropped)
	}
}

func TestABoundedBufferSurvivesOneWriteBiggerThanTheWholeCap(t *testing.T) {
	b := &boundedBuffer{max: 8}
	if _, err := b.Write([]byte("xy")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := b.Write([]byte("0123456789")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := b.String(); got != "23456789" {
		t.Errorf("the buffer holds %q, want the last eight bytes written", got)
	}

	if b.dropped != 4 {
		t.Errorf("the buffer dropped %d bytes, want 4 — two from the first write and two from the second", b.dropped)
	}
}

// TestABoundedBufferNeverGrowsPastItsCap is the whole point of the type. A
// hostile or merely long run must not be able to grow the Orbit process, so
// the array is checked and not only the length: the tail is slid down in
// place, and the only slack is the doubling Go's append does on the way up.
func TestABoundedBufferNeverGrowsPastItsCap(t *testing.T) {
	b := &boundedBuffer{max: 1024}
	for i := 0; i < 500; i++ {
		if _, err := b.Write(fill(97, byte('a'+i%26))); err != nil {
			t.Fatalf("Write: %v", err)
		}

		if len(b.buf) > b.max {
			t.Fatalf("after %d writes the buffer holds %d bytes, past its cap of %d", i+1, len(b.buf), b.max)
		}

		if cap(b.buf) > 2*b.max {
			t.Fatalf("after %d writes the array behind the buffer is %d bytes, which is growth the cap was meant to stop", i+1, cap(b.buf))
		}
	}

	if b.dropped != 500*97-1024 {
		t.Errorf("the buffer dropped %d bytes of %d written", b.dropped, 500*97)
	}
}

// TestTheResultObjectSurvivesACutStream is why the buffer keeps the end. A
// run long enough to hit the cap still has to report its session and its
// cost, or the cap would have quietly taken the two fields this adapter
// exists to capture.
func TestTheResultObjectSurvivesACutStream(t *testing.T) {
	b := &boundedBuffer{max: 2048}

	for i := 0; i < 200; i++ {
		line := fmt.Sprintf(`{"type":"assistant","text":%q}`+"\n", strings.Repeat("noise ", 8))
		if _, err := b.Write([]byte(line)); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	tail := `{"type":"result","result":"done","session_id":"9c1f8f2a","total_cost_usd":0.42}` + "\n"
	if _, err := b.Write([]byte(tail)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if b.dropped == 0 {
		t.Fatal("the test wrote less than the cap, so it is not testing a cut stream")
	}

	out, err := ParseStream(bytes.NewReader(b.Bytes()))
	if err != nil {
		t.Fatalf("ParseStream over a cut stream: %v", err)
	}

	if out.SessionID != "9c1f8f2a" || out.Cost != 0.42 {
		t.Errorf("a cut stream reported session %q and cost %v, losing what the cap was not meant to take", out.SessionID, out.Cost)
	}
}

// TestACutStreamSaysSoOnThePhasesOwnText holds the honest half of the bound.
// Truncation that announces itself is the rule internal/task's captured()
// set; an answer that lost its stream and reads like one that did not is the
// record stating something it cannot know.
func TestACutStreamSaysSoOnThePhasesOwnText(t *testing.T) {
	got := noteDropped("done", 4096)
	if !strings.Contains(got, "done") {
		t.Errorf("the note replaced the answer instead of following it: %q", got)
	}

	if !strings.Contains(got, "4096") {
		t.Errorf("the note does not say how much was dropped: %q", got)
	}

	if !strings.Contains(got, fmt.Sprint(maxStream)) {
		t.Errorf("the note does not say what the cap was: %q", got)
	}
}

func TestAnEmptyAnswerStillCarriesTheNote(t *testing.T) {
	got := noteDropped("", 512)
	if !strings.Contains(got, "512") {
		t.Errorf("a phase with no text and a cut stream said %q, which mentions neither", got)
	}

	if strings.HasPrefix(got, "\n") {
		t.Errorf("the note begins with a blank line it has nothing to separate from: %q", got)
	}
}

func TestAnUncutStreamSaysNothingExtra(t *testing.T) {
	if got := noteDropped("done", 0); got != "done" {
		t.Errorf("a stream that was never cut reported %q, want the answer untouched", got)
	}
}
