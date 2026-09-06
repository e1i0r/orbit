package supervisor

// Conversations, where they meet the store.
//
// The rule itself — what belongs to which conversation, what a title is,
// which ones were taken off the list — lives in internal/record, because
// internal/view folds the same thread for the screen and the two must not
// drift into two answers. What is here is the half that needs somewhere to
// write: which conversation a new line goes into, and taking one off the
// list.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Conversations is the thread grouped, most recently spoken in first.
func Conversations(events []record.Event) []record.Conversation {
	return record.Conversations(events)
}

// In is one conversation's turns, in the order they were said.
func In(id string, events []record.Event) []record.Event {
	return record.InConversation(id, events)
}

// NewID is the id a conversation started right now carries.
func NewID(at time.Time) string { return record.NewConversationID(at) }

// Remove takes a conversation out of the list and out of what the model is
// told, and leaves every line of it exactly where it is.
func Remove(s *store.Store, id string) error {
	if s == nil {
		return fmt.Errorf("store cannot be nil")
	}

	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.AppendMessage(record.Event{
		At:   time.Now().UTC(),
		Kind: record.SupervisorConversationRemoved,
		Data: map[string]string{"conversation": id},
	})
}
