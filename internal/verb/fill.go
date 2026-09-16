package verb

// Reading the rest of a line into the fields a verb takes.
//
// It is here rather than in whichever way in asked, because the rules are
// subtle and a second copy of them is a second grammar: the order is the
// order the verb declares, one word per field it must have, and the field of
// words takes everything left over. A reader who learned `orbit note abc it
// needs a test` at a terminal types the same thing into a chat and means it.

import "strings"

// Fill takes what is left of a line and puts it in the fields it is for.
//
// `orbit settings set autopilot on` is what a person writes; `-key autopilot
// -value on` is what a form would have wanted. The order is the order the
// verb declares, one word per field it must have, and a field of words takes
// everything that is left — so the sentence at the end of `orbit note abc it
// needs a test` arrives whole. A flag still wins, because a script that
// spelled it out meant it.
//
// A leading "--" is dropped. It is the shell's way of saying the flags are
// over, and flag stops at the first positional — so a caller that puts it
// after the id is saying so once the flags are over already. Kept, it
// became the first word of every note the window wrote.
func Fill(v Verb, args map[string]string, rest []string) []string {
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}

	for _, f := range v.Takes {
		if len(rest) == 0 {
			return nil
		}

		if args[f.Name] != "" {
			continue
		}

		if f.Kind == Words {
			args[f.Name] = strings.Join(rest, " ")

			return nil
		}

		// An optional name is still a name when it is there: `reconcile
		// PAY-1` narrows the sweep to one task, and a reader who typed it
		// named something rather than adding noise. Yes-or-no stays on
		// its flag — a bare "true" on the line is a typo until proven
		// otherwise — and a field of words still takes the rest.
		if !f.Needed && f.Kind != Named {
			continue
		}

		args[f.Name], rest = rest[0], rest[1:]
	}

	// Whatever is left has nowhere to go. Printing the settings for a
	// reader who typed `orbit settings autopilot on` and walking away is
	// how a run sat at a gate for ten minutes waiting for an autopilot
	// nobody had turned on.
	return rest
}
