package verb

// onOff is how a switch is written on the command line.
//
// The vocabulary is the window's, not the config file's: "on" and "off" first,
// because that is what the switch on screen is labelled and a command line
// that disagreed with it would be two vocabularies for one setting. ParseBool
// comes after, so true/false/1/0 are accepted rather than refused — but the
// assertion worth making is the refusal itself, and that it names the setting.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// TestOnOffReadsTheWordsTheWindowUses.
func TestOnOffReadsTheWordsTheWindowUses(t *testing.T) {
	p := words.For("en")

	cases := []struct {
		value string
		want  bool
	}{
		{"on", true},
		{"off", false},
		// Case and spacing are what a person actually types.
		{"ON", true},
		{"Off", false},
		{"  on  ", true},
		// What anybody who has used a config file expects.
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	}

	for _, c := range cases {
		got, err := onOff(p, "autopilot", c.value)
		if err != nil {
			t.Errorf("%q was refused: %v", c.value, err)

			continue
		}

		if got != c.want {
			t.Errorf("%q read as %v, want %v", c.value, got, c.want)
		}
	}
}

// TestAWordThatIsNeitherIsRefusedNamingTheSetting.
//
// The refusal has to say which setting it was about. A reader who typed
// `orbit set autopilot yes` and was answered "yes is not a boolean" has to go
// and work out which of the settings they meant; the one that names
// autopilot does not.
func TestAWordThatIsNeitherIsRefusedNamingTheSetting(t *testing.T) {
	p := words.For("en")

	on, err := onOff(p, "autopilot", "yes")
	if err == nil {
		t.Fatalf("%q was accepted as %v", "yes", on)
	}

	if on {
		t.Error("a refused value came back as on")
	}

	said := err.Error()

	if !strings.Contains(said, "autopilot") {
		t.Errorf("the refusal did not name the setting: %q", said)
	}

	if !strings.Contains(said, "yes") {
		t.Errorf("the refusal did not carry back the value that was refused: %q", said)
	}
}

// TestNothingAtAllIsRefused.
//
// An empty value is the one a reader reaches by pressing enter at a prompt,
// and ParseBool refuses it — so it has to come back as a sentence rather than
// as a silent "off".
func TestNothingAtAllIsRefused(t *testing.T) {
	if _, err := onOff(words.For("en"), "check-record", ""); err == nil {
		t.Error("an empty value was accepted")
	}
}

// TestOffOnIsTheWordTheConfirmationPrints.
//
// The other half of the same vocabulary: a switch that accepts "on" and then
// confirms with "true" is two vocabularies for one setting, which is the thing
// onOff exists to stop.
func TestOffOnIsTheWordTheConfirmationPrints(t *testing.T) {
	if got := offOn(true); got != "on" {
		t.Errorf("on printed as %q, want \"on\"", got)
	}

	if got := offOn(false); got != "off" {
		t.Errorf("off printed as %q, want \"off\"", got)
	}
}
