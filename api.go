package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// apiBaseURL is the base URL for the Pocket Casts API.
// It's a var (not const) so tests can point it at an httptest server.
var apiBaseURL = "https://api.pocketcasts.com"

// LoginRequest is the JSON body sent to the login endpoint.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Scope    string `json:"scope"`
}

// LoginResponse is the JSON body returned by the login endpoint.
type LoginResponse struct {
	Token string `json:"token"`
	UUID  string `json:"uuid"`
	Email string `json:"email"`
}

// Episode represents a single episode from the history response.
//
// We use `json:"..."` tags to map JSON keys to Go field names.
// The Status field is not part of the API response — it's derived from
// PlayedUpTo and Duration after fetching.
type Episode struct {
	UUID         string `json:"uuid"`
	Title        string `json:"title"`
	PodcastTitle string `json:"podcastTitle"`
	PodcastUUID  string `json:"podcastUuid"`
	URL          string `json:"url"`
	Published    string `json:"published"`
	Duration     int    `json:"duration"`
	PlayedUpTo      int    `json:"playedUpTo"`
	Status          string `json:"status"`
	ProgressPercent int    `json:"progress_percent"`
}

// EpisodeStatus derives a listening status from playback progress.
//
// Thresholds: "completed" if ≥90% played (Pocket Casts often stops a few
// seconds before the end), "started" if any progress, "unplayed" otherwise.
func EpisodeStatus(playedUpTo, duration int) string {
	if duration <= 0 || playedUpTo <= 0 {
		return "unplayed"
	}
	if float64(playedUpTo)/float64(duration) >= 0.9 {
		return "completed"
	}
	return "started"
}

// EpisodeProgressPercent computes playback progress as a percentage (0–100).
func EpisodeProgressPercent(playedUpTo, duration int) int {
	if duration <= 0 || playedUpTo <= 0 {
		return 0
	}
	pct := playedUpTo * 100 / duration
	if pct > 100 {
		return 100
	}
	return pct
}

// HistoryResponse is the JSON body returned by the history endpoint.
type HistoryResponse struct {
	Episodes []Episode `json:"episodes"`
	Total    int       `json:"total"`
}

// Client is the Pocket Casts API client.
//
// Storing the http.Client allows callers to customise timeouts, transport,
// etc. This is a common Go pattern — accept an *http.Client rather than
// creating one internally.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
}

// NewClient creates a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		HTTPClient: http.DefaultClient,
		BaseURL:    apiBaseURL,
	}
}

// Login authenticates with Pocket Casts and returns a bearer token.
//
// The pattern of building a request, executing it, and decoding the response
// is the standard way to call HTTP APIs in Go. We use json.NewEncoder and
// json.NewDecoder to stream JSON rather than marshaling to intermediate
// byte slices.
func (c *Client) Login(email, password string) (string, error) {
	body := LoginRequest{
		Email:    email,
		Password: password,
		Scope:    "webplayer",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling login request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/user/login", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending login request: %w", err)
	}
	// defer ensures the body is closed when the function returns, even on
	// early error returns. This prevents resource leaks — a very common Go
	// pattern.
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, respBody)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decoding login response: %w", err)
	}

	if loginResp.Token == "" {
		return "", fmt.Errorf("login response contained empty token")
	}

	return loginResp.Token, nil
}

// fetchHistoryBody makes the history API call and returns the raw response body.
func (c *Client) fetchHistoryBody(token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/user/history", nil)
	if err != nil {
		return nil, fmt.Errorf("creating history request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending history request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading history response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("history request failed (HTTP %d): %s", resp.StatusCode, body)
	}

	return body, nil
}

// FetchHistory retrieves the user's listening history.
//
// The token parameter is the bearer token obtained from Login. Each episode's
// Status field is derived from its playback progress after decoding.
func (c *Client) FetchHistory(token string) ([]Episode, error) {
	body, err := c.fetchHistoryBody(token)
	if err != nil {
		return nil, err
	}

	var historyResp HistoryResponse
	if err := json.Unmarshal(body, &historyResp); err != nil {
		return nil, fmt.Errorf("decoding history response: %w", err)
	}

	for i := range historyResp.Episodes {
		ep := &historyResp.Episodes[i]
		ep.Status = EpisodeStatus(ep.PlayedUpTo, ep.Duration)
		ep.ProgressPercent = EpisodeProgressPercent(ep.PlayedUpTo, ep.Duration)
	}

	return historyResp.Episodes, nil
}

// FetchHistoryRaw retrieves the raw JSON response from the history endpoint.
//
// Useful for inspecting which fields the API actually returns, since our
// Episode struct only maps a subset of them.
func (c *Client) FetchHistoryRaw(token string) ([]byte, error) {
	return c.fetchHistoryBody(token)
}
