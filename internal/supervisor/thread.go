package supervisor

// The thread itself: one table under the state root, appended to and read
// back, and the one way a turn is taken out of it again.
//
// It is a table of its own rather than a task's events because it belongs to
// no task, and hanging it off one would mean inventing a task for it. What
// comes back out is still record.Event, so internal/view folds the thread
// exactly as it folded the file — where the turns are kept changed, what a
// turn is did not.

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Record appends an event to the global supervisor conversation log.
func Record(s *store.Store, kind, by, channel, taskID, repo, text string) error {
	if s == nil {
		return fmt.Errorf("store cannot be nil")
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("supervisor message cannot be empty")
	}

	if kind == "" {
		kind = record.SupervisorMessage
	}

	data := map[string]string{}
	if by = strings.TrimSpace(by); by != "" {
		data["by"] = by
	}

	if channel = strings.TrimSpace(channel); channel != "" {
		data["channel"] = channel
	}

	if taskID = strings.TrimSpace(taskID); taskID != "" {
		data["task_id"] = taskID
	}

	if repo = strings.TrimSpace(repo); repo != "" {
		data["repo"] = repo
	}

	e := record.Event{
		At:   time.Now().UTC(),
		Kind: kind,
		Text: text,
		Data: data,
	}

	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.AppendMessage(e)
}

// Events reads the whole global supervisor thread, oldest first.
func Events(s *store.Store) ([]record.Event, error) {
	if s == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	return d.Messages()
}

// Retract takes back one turn of the supervisor thread.
//
// Nothing is erased and nothing pretends to be. A retraction is another line
// appended after the one it withdraws, naming it by record.Stamp: the thread
// still shows that the sentence was said, and stops putting it in front of
// the model. That is the only shape this can take in a log that is
// append-only, and it is also the honest one — a message somebody regrets is
// still a message they sent, and a reader working out why the supervisor
// concluded something needs to see it.
//
// A timestamp nothing was written at is refused rather than appended. A
// retraction that matches no line is a typo, and a log that quietly accepts
// one leaves somebody believing they took something back.
func Retract(s *store.Store, at time.Time) error {
	if s == nil {
		return fmt.Errorf("store cannot be nil")
	}

	if at.IsZero() {
		return fmt.Errorf("a retraction has to name the turn it takes back")
	}

	events, err := Events(s)
	if err != nil {
		return err
	}

	want := record.Stamp(at)
	for _, e := range events {
		if e.Kind == record.SupervisorRetracted || record.Stamp(e.At) != want {
			continue
		}

		d, err := s.Record()
		if err != nil {
			return err
		}

		return d.AppendMessage(record.Event{
			At:   time.Now().UTC(),
			Kind: record.SupervisorRetracted,
			Data: map[string]string{"at": want},
		})
	}

	return fmt.Errorf("nothing in the supervisor thread was written at %s", want)
}
