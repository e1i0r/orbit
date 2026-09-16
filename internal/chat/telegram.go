package chat

// Telegram, as a Channel.
//
// The first service adapter, and the shape every other one follows: it knows
// about tokens, polling and what this particular service does to text, and
// it knows nothing about verbs. What arrives is a Message and what leaves is
// a string; everything between those two is the desk's.
//
// Long-poll and not a webhook, which is the decision that makes the whole
// thing work from a laptop. getUpdates is an outbound HTTPS request that
// waits — so there is no port to open, no public address to have, no
// certificate to hold and nothing of yours reachable from the internet.
// Telegram does the push to the phone, which is the expensive half and the
// one nobody should be operating themselves.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

// waitFor is how long one poll hangs before Telegram answers with nothing.
//
// Long on purpose: a poll that returns at once is a loop doing nothing at a
// hundred requests a minute, and the connection sitting open costs nobody
// anything. Fifty seconds leaves room under the client's own timeout.
const waitFor = 50

// Telegram is the bot, and the conversation it is allowed to have.
type Telegram struct {
	token string
	http  *http.Client
	// offset is the first update not yet seen. Telegram keeps an update
	// until it is acknowledged by asking for the one after it, so this is
	// what stops a message being answered twice — and what makes a crash
	// mid-answer cost a repeat rather than a loss.
	offset int
}

// Bot is a channel over one bot token.
func Bot(token string) *Telegram {
	return &Telegram{
		token: token,
		// Longer than the poll it holds open, or every poll ends as a
		// client timeout and the log fills with failures of nothing.
		http: &http.Client{Timeout: (waitFor + 15) * time.Second},
	}
}

// Name is what the record and the log call it.
func (t *Telegram) Name() string { return "telegram" }

// Listen hands over every message until the context is done.
//
// It never returns an error for a service that is merely unreachable. Orbit
// is a program somebody leaves running, and a channel that ended the loop
// because a laptop closed its lid is a channel that is silently off for the
// rest of the afternoon.
func (t *Telegram) Listen(ctx context.Context, said func(Message)) error {
	for {
		// The reader leaving is how this ends, and it is not a failure of
		// the service.
		if err := ctx.Err(); err != nil {
			return nil //nolint:nilerr // deliberate: see above
		}

		updates, err := t.poll(ctx)
		if err != nil {
			if stopped := ctx.Err(); stopped != nil {
				return nil //nolint:nilerr // the poll failed because it was cancelled
			}

			logger.Warn("chat/telegram", "poll: %v", err)
			// A pause before trying again, so an outage is a request a
			// second rather than a request a microsecond.
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
			}

			continue
		}

		for _, u := range updates {
			t.offset = u.UpdateID + 1

			if u.Message.Text == "" {
				continue
			}

			said(Message{
				Where: strconv.FormatInt(u.Message.Chat.ID, 10),
				Who:   strconv.FormatInt(u.Message.From.ID, 10),
				Text:  u.Message.Text,
			})
		}
	}
}

// Say answers into one conversation.
//
// Twice, if the first one is refused. MarkdownV2 is strict — one unescaped
// character out of eighteen and the whole message is rejected — and what
// this carries is not all Orbit's to escape: the supervisor answers in
// markdown because it was asked to, and escaping that would be deleting the
// formatting rather than protecting it.
//
// So the formatted send is attempted and a refusal falls back to plain text.
// A stray character then costs the formatting of one message instead of the
// message itself, which is the trade worth making: an answer nobody sees is
// worse than an answer with an asterisk in it.
func (t *Telegram) Say(ctx context.Context, where, text string) error {
	if err := t.send(ctx, where, asMarkdown(text), "MarkdownV2"); err == nil {
		return nil
	}

	return t.send(ctx, where, plain(text), "")
}

// send is one attempt, in one markup.
func (t *Telegram) send(ctx context.Context, where, text, markup string) error {
	message := map[string]any{"chat_id": where, "text": text}
	if markup != "" {
		message["parse_mode"] = markup
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode the message: %w", err)
	}

	res, err := t.post(ctx, "sendMessage", body)
	if err != nil {
		return err
	}

	return res.Body.Close()
}

// Working shows "typing…" in the conversation.
//
// Telegram stops showing it after about five seconds or when a message
// arrives, whichever comes first, which is exactly the shape this wants: the
// indicator ends by itself when the answer lands, and nothing has to be
// taken back down.
func (t *Telegram) Working(ctx context.Context, where string) error {
	body, err := json.Marshal(map[string]any{"chat_id": where, "action": "typing"})
	if err != nil {
		return fmt.Errorf("encode the action: %w", err)
	}

	res, err := t.post(ctx, "sendChatAction", body)
	if err != nil {
		return err
	}

	return res.Body.Close()
}

// update is one thing Telegram had waiting.
type update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

// poll asks for everything since the last update it saw.
func (t *Telegram) poll(ctx context.Context) ([]update, error) {
	body, err := json.Marshal(map[string]any{
		"offset":  t.offset,
		"timeout": waitFor,
		// Only messages. A bot is told about edits, joins and a dozen other
		// things it has no opinion about, and asking for them is asking to
		// decide what to do with them.
		"allowed_updates": []string{"message"},
	})
	if err != nil {
		return nil, fmt.Errorf("encode the poll: %w", err)
	}

	res, err := t.post(ctx, "getUpdates", body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close() //nolint:errcheck // a read-only body on the way out

	var answer struct {
		OK          bool     `json:"ok"`
		Result      []update `json:"result"`
		Description string   `json:"description"`
	}

	if err := json.NewDecoder(res.Body).Decode(&answer); err != nil {
		return nil, fmt.Errorf("read what telegram answered: %w", err)
	}

	if !answer.OK {
		return nil, fmt.Errorf("telegram refused: %s", answer.Description)
	}

	return answer.Result, nil
}

// post is one call to the bot API.
func (t *Telegram) post(ctx context.Context, method string, body []byte) (*http.Response, error) {
	// The token is in the path, which is how this API is built. It is never
	// logged: every error below names the method and not the URL.
	url := "https://api.telegram.org/bot" + t.token + "/" + method

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build the %s request: %w", method, err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close() //nolint:errcheck // the status is the answer

		return nil, fmt.Errorf("%s answered %s", method, res.Status)
	}

	return res, nil
}

// asMarkdown is an answer in the markup this service reads.
//
// A fence Reply put around a listing stays a fence — it is already what
// MarkdownV2 spells a code block with — and what is inside one needs only
// the backslash and the backtick escaped. Outside, this package's own
// emphasis becomes the asterisks the service wants.
//
// What is deliberately not escaped is everything else. Telegram's own rule
// is that eighteen characters must be, and obeying it here would mean
// escaping the supervisor's markdown out of existence: it answers in
// markdown because it was asked to. Say sends this first and falls back to
// plain text if the service refuses it, which is what makes the gamble
// affordable.
func asMarkdown(text string) string {
	var b strings.Builder

	for i, part := range strings.Split(text, "```") {
		// Odd parts are what the fences held.
		if i%2 == 1 {
			b.WriteString("```\n" + inCode(strings.Trim(part, "\n")) + "\n```")

			continue
		}

		b.WriteString(strings.NewReplacer(Strong, "*", endStrong, "*").Replace(part))
	}

	return b.String()
}

// inCode is the two characters a fenced block still has to escape.
func inCode(text string) string {
	return strings.NewReplacer("\\", "\\\\", "`", "\\`").Replace(text)
}

// plain is the same answer with the markup taken out, for the send that
// follows one the service refused.
func plain(text string) string {
	return strings.NewReplacer(Strong, "", endStrong, "", "```", "").Replace(text)
}
