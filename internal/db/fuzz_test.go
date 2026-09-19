package db

// The column an event's data lives in, from both sides.
//
// The bytes on the way out are this package's own, and the bytes on the way
// back in are whatever the file holds — an older Orbit's, a row somebody
// edited, a file half written when the machine went down. Nothing here may
// panic over any of them: the record is the one thing that cannot be rebuilt,
// so a reader that falls over on one row takes the whole history with it.

import (
	"maps"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestWhatGoesIntoTheColumnComesBackTheSame, for the shapes a run actually
// writes: the empty map that most events carry, a field with a newline in
// it, and text that is not the writer's own language.
func TestWhatGoesIntoTheColumnComesBackTheSame(t *testing.T) {
	for _, one := range []struct {
		why  string
		data map[string]string
	}{
		{"most events carry none", nil},
		{"and an empty map is the same as none", map[string]string{}},
		{"an ordinary field", map[string]string{"by": "operator"}},
		{"several", map[string]string{"by": "operator", "phase": "implement", "n": "2"}},
		{"a value with a line break in it", map[string]string{"error": "first\nsecond"}},
		{"a value with quotes and braces", map[string]string{"args": `{"command":"echo \"hi\""}`}},
		{"a value that is not English", map[string]string{"text": "un árbol con acentos"}},
		{"a value that is not a language at all", map[string]string{"text": "🛰 ✅"}},
		{"an empty value, which is not the same as no field", map[string]string{"why": ""}},
	} {
		column, err := encode(one.data)
		if err != nil {
			t.Errorf("%s: encode: %v", one.why, err)
			continue
		}

		back, err := decode(column)
		if err != nil {
			t.Errorf("%s: decode %q: %v", one.why, column, err)
			continue
		}

		if len(one.data) == 0 {
			if back != nil {
				t.Errorf("%s: came back as %v, want nothing", one.why, back)
			}

			continue
		}

		if !maps.Equal(back, one.data) {
			t.Errorf("%s: %v came back as %v", one.why, one.data, back)
		}
	}
}

// FuzzEventData is the column read against bytes nothing here wrote.
//
// Two claims. Anything this package encodes comes back equal to what went
// in — that is the round trip, and it is what says a run's own data survives
// a restart. And anything else either decodes to a map or is refused: never
// a panic, and never a half-read map beside an error, because a caller that
// was handed both would write the half back.
func FuzzEventData(f *testing.F) {
	for _, seed := range []string{
		"", "{}", `{"by":"operator"}`, `{"a":"b","c":"d"}`,
		"null", "[]", `{"a":1}`, `{"a":null}`, `{"a":["b"]}`,
		"{", `{"a":`, `{"a":"\ud800"}`, "\x00", strings.Repeat(`{"a":"b"},`, 64),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		got, err := decode(raw)
		if err != nil {
			if got != nil {
				t.Errorf("decode(%q) refused and still answered %v", raw, got)
			}

			return
		}

		// What came back is a map this package could write again, and the
		// second column is the same as the first: a round trip that moved
		// would mean the record says something different every time it is
		// copied.
		again, err := encode(got)
		if err != nil {
			t.Fatalf("what decode(%q) answered cannot be encoded: %v", raw, err)
		}

		twice, err := decode(again)
		if err != nil {
			t.Fatalf("what encode answered cannot be decoded: %v", err)
		}

		if !maps.Equal(twice, got) {
			t.Errorf("decode(%q) = %v, and round-tripping it gives %v", raw, got, twice)
		}

		// Every key and value that came back is text a column can hold,
		// because the column is TEXT and SQLite will not take bytes that
		// are not.
		for k, v := range got {
			if !utf8.ValidString(k) || !utf8.ValidString(v) {
				t.Errorf("decode(%q) answered a key or value that is not text: %q = %q", raw, k, v)
			}
		}
	})
}
