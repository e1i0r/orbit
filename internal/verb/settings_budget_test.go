package verb

// The three numbers somebody types at the settings table: two budgets in
// dollars and a floor as a percentage.
//
// Every one of them has a zero that means something — no budget, no floor —
// so the refusals have to be read at their own edges. A zero refused is a
// reader who cannot turn a cap off; a hundred accepted is a queue that never
// starts again and says only that the floor is on.

import (
	"strings"
	"testing"
)

// set is one key written through the verb, answering what it was told.
func set(t *testing.T, w *testWorld, key, value string) Out {
	t.Helper()

	return mustAsk(t, w, "settings set", In{
		Args: map[string]string{"key": key, "value": value}, By: "operator",
	})
}

// TestABudgetIsWrittenDownAndReadBackAsItWasTyped.
//
// The amount is money, so it comes back to the pound rather than rounded to
// something near it: a reader who set a cap of a dollar twenty-five and is
// shown a dollar twenty has been told their cap is something else.
func TestABudgetIsWrittenDownAndReadBackAsItWasTyped(t *testing.T) {
	w := worldOf(t)

	for _, one := range []struct{ key, value string }{
		{"budget-task", "1.25"},
		{"budget-workspace", "40.5"},
	} {
		out := set(t, w, one.key, one.value)
		if !strings.Contains(out.Said, one.value) {
			t.Errorf("setting %s to %s answered %q", one.key, one.value, out.Said)
		}

		table := mustAsk(t, w, "settings", In{By: "operator"})
		if !strings.Contains(table.Said, one.value) {
			t.Errorf("the table reads back %s as something else:\n%s", one.key, table.Said)
		}
	}
}

// TestZeroIsHowABudgetIsTurnedOff, which is why it is the one number below
// the refusal that has to be accepted: every field of the settings file has
// a working zero, and for a cap the working zero is that there is none.
func TestZeroIsHowABudgetIsTurnedOff(t *testing.T) {
	w := worldOf(t)

	for _, key := range []string{"budget-task", "budget-workspace"} {
		out := set(t, w, key, "0")
		if !strings.Contains(out.Said, "0") {
			t.Errorf("setting %s to nothing answered %q", key, out.Said)
		}

		mustRefuse(t, w, "settings set", In{
			Args: map[string]string{"key": key, "value": "-1"}, By: "operator",
		})

		mustRefuse(t, w, "settings set", In{
			Args: map[string]string{"key": key, "value": "a dollar"}, By: "operator",
		})
	}
}

// TestAQuotaFloorIsAPercentageBetweenNoneAndNinetyNine.
//
// A hundred would hold the queue for ever and below zero is not a share of
// anything, so both ends are refused where they are typed — and the two
// numbers just inside them are accepted, because a floor of none is how the
// reader turns it off and ninety-nine is the strictest one they can mean.
func TestAQuotaFloorIsAPercentageBetweenNoneAndNinetyNine(t *testing.T) {
	w := worldOf(t)

	for _, value := range []string{"0", "99", "50"} {
		out := set(t, w, "quota-floor", value)
		if !strings.Contains(out.Said, value) {
			t.Errorf("a floor of %s answered %q", value, out.Said)
		}
	}

	for _, value := range []string{"100", "-1", "101", "half"} {
		mustRefuse(t, w, "settings set", In{
			Args: map[string]string{"key": "quota-floor", "value": value}, By: "operator",
		})
	}
}

// TestATimeoutIsWrittenDownTheWayItWasTyped, and read back as a length of
// time rather than as a number of nanoseconds.
func TestATimeoutIsWrittenDownTheWayItWasTyped(t *testing.T) {
	w := worldOf(t)

	// Read back the way they were written: the screen offers 30m, 1h and
	// 2h as pills, and a value spelled "2h0m0s" is a dial with nothing
	// chosen on it.
	for _, one := range []struct{ typed, back string }{
		{"45m", "45m"},
		{"2h", "2h"},
		{"90s", "90s"},
	} {
		out := set(t, w, "run-timeout", one.typed)
		if !strings.Contains(out.Said, one.back) {
			t.Errorf("setting run-timeout to %s answered %q, want %s", one.typed, out.Said, one.back)
		}

		table := mustAsk(t, w, "settings", In{By: "operator"})
		if !strings.Contains(table.Said, one.back) {
			t.Errorf("the table reads back the timeout as something else:\n%s", table.Said)
		}
	}
}

// TestZeroIsHowATimeoutIsTurnedOff. Every field of the settings file has a
// working zero, and for a limit the working zero is that there is none — so
// "0" has to be something a reader can type, and the table has to say it
// back in the same word they would type to turn it off again.
func TestZeroIsHowATimeoutIsTurnedOff(t *testing.T) {
	w := worldOf(t)

	set(t, w, "run-timeout", "2h")

	out := set(t, w, "run-timeout", "0")
	if !strings.Contains(out.Said, "0") {
		t.Errorf("turning the timeout off answered %q", out.Said)
	}

	// The stored form and not "2h": the sentence beside the row says
	// "as 45m or 2h", and a test looking for that in the whole table is
	// reading the help rather than the value.
	table := mustAsk(t, w, "settings", In{By: "operator"})
	if strings.Contains(table.Said, "  2h ") {
		t.Errorf("the timeout was turned off and the table still holds it:\n%s", table.Said)
	}

	// A negative deadline is a run that has already run out, and a word
	// that is not a length of time is a typo somebody wants told about.
	mustRefuse(t, w, "settings set", In{
		Args: map[string]string{"key": "run-timeout", "value": "-5m"}, By: "operator",
	})

	mustRefuse(t, w, "settings set", In{
		Args: map[string]string{"key": "run-timeout", "value": "a while"}, By: "operator",
	})
}
