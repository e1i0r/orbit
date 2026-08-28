package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// linearEndpoint is where the GraphQL query goes. It is a variable so a test
// can point it at a server that answers the way Linear does.
var linearEndpoint = "https://api.linear.app/graphql"

// FetchLinear fetches issue title and description from the Linear GraphQL API.
func FetchLinear(ctx context.Context, apiKey, id string) (string, string, error) {
	apiKey = strings.TrimSpace(apiKey)

	id = strings.TrimSpace(id)
	if apiKey == "" || id == "" {
		return "", "", errors.New("apiKey and id are required")
	}

	query := `query GetIssue($id: String!) {
		issue(id: $id) {
			title
			description
		}
	}`

	reqBody, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]string{"id": id},
	})
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		linearEndpoint,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", "", fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("linear api request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("linear api status %d", resp.StatusCode)
	}

	// GraphQL answers 200 to a request it refused. A revoked API key, an
	// issue on another workspace and a malformed query all come back as
	// HTTP 200 with data.issue null and the reason in errors — which this
	// did not read, so all three returned ("", "", nil): an issue with no
	// title, no description and no error, and the task went on to be
	// created from it. The status code check above only ever caught the
	// network being down.
	var res struct {
		Data struct {
			Issue *struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"issue"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", fmt.Errorf("decode linear response: %w", err)
	}

	if len(res.Errors) > 0 {
		return "", "", fmt.Errorf("linear api: %s", res.Errors[0].Message)
	}

	if res.Data.Issue == nil {
		return "", "", fmt.Errorf("linear has no issue %s, or this API key cannot see it", id)
	}

	return res.Data.Issue.Title, res.Data.Issue.Description, nil
}
