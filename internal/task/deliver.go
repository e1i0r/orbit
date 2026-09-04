package task

// What the cockpit's delivery keys asked for, written down on the task they
// were pressed about.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Delivering records that a delivery verb was asked for.
//
// It is written at the press and not when the work lands, because the work
// mostly lands elsewhere: six of these verbs are handed to the supervisor,
// which runs an engine that answers minutes later into a thread of its own.
// Until this event existed, a reader who pressed a key on a task saw the
// band change once and the task itself never mention it — which is
// indistinguishable from a key that does nothing.
//
// verb is the caption the key was offered under, so the record says what the
// reader read rather than the name of whatever carries it: fix checks and
// more tests were both `orbit note` at one point, and a record naming the
// command answers a question nobody asked.
//
// by is what was handed the work — "supervisor", or the command's own name.
func Delivering(s *store.Store, t Task, verb, by string) error {
	verb = strings.TrimSpace(verb)
	if verb == "" {
		return fmt.Errorf("task %s: a delivery needs the verb it was asked under", t.ID)
	}

	e := record.Event{Kind: record.DeliverAsked, Data: map[string]string{"verb": verb}}
	if by = strings.TrimSpace(by); by != "" {
		e.Data["by"] = by
	}

	return emit(s, t, e)
}

// Delivered records how one of them ended: what came back, and the reason it
// broke where it did.
//
// The two travel together rather than in two kinds, for the reason a phase's
// failure keeps whatever the engine printed: an answer cut off halfway is
// still the only account there is of what was being done. A failure with
// nothing said and nothing to say is still written — that a verb came back
// at all is what tells a reader it is no longer out there working.
func Delivered(s *store.Store, t Task, verb, text string, failure error) error {
	verb = strings.TrimSpace(verb)
	if verb == "" {
		return fmt.Errorf("task %s: a delivery needs the verb it was asked under", t.ID)
	}

	e := record.Event{
		Kind: record.DeliverAnswered,
		Text: strings.TrimSpace(text),
		Data: map[string]string{"verb": verb},
	}
	if failure != nil {
		e.Data["error"] = failure.Error()
	}

	return emit(s, t, e)
}
