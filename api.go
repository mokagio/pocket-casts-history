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
// Fields use pointer types or omitempty where the API may omit them.
type Episode struct {
	UUID         string `json:"uuid"`
	Title        string `json:"title"`
	PodcastTitle string `json:"podcastTitle"`
	PodcastUUID  string `json:"podcastUuid"`
	URL          string `json:"url"`
	Published    string `json:"published"`
	Duration     int    `json:"duration"`
	PlayedUpTo   int    `json:"playedUpTo"`
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

// FetchHistory retrieves the user's listening history.
//
// The token parameter is the bearer token obtained from Login.
func (c *Client) FetchHistory(token string) ([]Episode, error) {
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

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("history request failed (HTTP %d): %s", resp.StatusCode, respBody)
	}

	var historyResp HistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
		return nil, fmt.Errorf("decoding history response: %w", err)
	}

	return historyResp.Episodes, nil
}
