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
		"https://api.linear.app/graphql",
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

	var res struct {
		Data struct {
			Issue struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"issue"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", fmt.Errorf("decode linear response: %w", err)
	}
	return res.Data.Issue.Title, res.Data.Issue.Description, nil
}
