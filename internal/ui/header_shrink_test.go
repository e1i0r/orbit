package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// lesserFields are the header's right-hand fields other than the brake, by
// their chips: the upgrade notice, the repository count, the engine and the
// language.
const lesserFields = "✨📦🧠🌐"

// TestTheBrakeIsTheLastHeaderFieldGivenUp.
//
// headerLine's own comment argues the order the fields go in, and the code
// went the other way: the drop took the last field in the slice and
// headerFields appends the brake last, so at 95 cells the header gave up the
// one field that says why nothing is starting and kept an upgrade notice —
// a command that will still be there tomorrow — for another forty-five.
//
// The walk is every width rather than a handful, because the field that goes
// changes at whatever width the one before it stopped fitting, and those
// widths are not round numbers.
func TestTheBrakeIsTheLastHeaderFieldGivenUp(t *testing.T) {
	m, _ := testModel(t, 200, 30)
	m.opts.Root = "~/work/acme/payments"
	m.opts.Settings = &settings{autopilot: true, lang: "en", unread: 1}
	m.upgradeAvailable = "1.2.3"

	if line := ansi.Strip(m.headerLine(200)); !strings.Contains(line, "⚠️") || !strings.ContainsAny(line, lesserFields) {
		t.Fatalf("the header at 200 cells is %q, and this test needs every field on it", line)
	}

	alone := false

	for w := 200; w >= 12; w-- {
		line := ansi.Strip(m.headerLine(w))
		if !strings.Contains(line, "⚠️") {
			if strings.ContainsAny(line, lesserFields) {
				t.Fatalf("at %d cells the header gave up the brake and kept a field it can do without: %q", w, line)
			}

			continue
		}

		alone = alone || !strings.ContainsAny(line, lesserFields)
	}

	if !alone {
		t.Error("no width drew the brake on its own, so it was never the last field standing")
	}
}
