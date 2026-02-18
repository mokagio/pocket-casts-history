package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLogin verifies the login flow against a fake HTTP server.
//
// httptest.NewServer starts a real HTTP server on a random port, backed by
// the handler we provide. This is the standard Go way to test HTTP clients
// without mocking — you get a real TCP connection, real HTTP parsing, etc.
func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantToken  string
		wantErr    bool
	}{
		{
			name: "successful login",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify the request is well-formed.
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected application/json content type")
				}

				var body LoginRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decoding request body: %v", err)
				}
				if body.Email != "test@example.com" {
					t.Errorf("expected email test@example.com, got %s", body.Email)
				}
				if body.Scope != "webplayer" {
					t.Errorf("expected scope webplayer, got %s", body.Scope)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(LoginResponse{
					Token: "fake-token-123",
					UUID:  "user-uuid",
					Email: "test@example.com",
				})
			},
			wantToken: "fake-token-123",
		},
		{
			name: "invalid credentials",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid credentials"}`))
			},
			wantErr: true,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "empty token in response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(LoginResponse{Token: ""})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// httptest.NewServer gives us a URL like "http://127.0.0.1:12345"
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			token, err := client.Login("test@example.com", "password123")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Login() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && token != tt.wantToken {
				t.Errorf("Login() = %q, want %q", token, tt.wantToken)
			}
		})
	}
}

// TestEpisodeStatus verifies the status derivation from playback progress.
func TestEpisodeStatus(t *testing.T) {
	tests := []struct {
		name       string
		playedUpTo int
		duration   int
		want       string
	}{
		{name: "no progress", playedUpTo: 0, duration: 3600, want: "unplayed"},
		{name: "just started", playedUpTo: 51, duration: 3778, want: "started"},
		{name: "halfway", playedUpTo: 1800, duration: 3600, want: "started"},
		{name: "exactly 90%", playedUpTo: 900, duration: 1000, want: "completed"},
		{name: "above 90%", playedUpTo: 3500, duration: 3600, want: "completed"},
		{name: "fully played", playedUpTo: 3600, duration: 3600, want: "completed"},
		{name: "zero duration", playedUpTo: 100, duration: 0, want: "unplayed"},
		{name: "both zero", playedUpTo: 0, duration: 0, want: "unplayed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EpisodeStatus(tt.playedUpTo, tt.duration)
			if got != tt.want {
				t.Errorf("EpisodeStatus(%d, %d) = %q, want %q", tt.playedUpTo, tt.duration, got, tt.want)
			}
		})
	}
}

// TestFetchHistory verifies the history endpoint against a fake server.
func TestFetchHistory(t *testing.T) {
	sampleEpisodes := []Episode{
		{
			UUID:         "ep-1",
			Title:        "First Episode",
			PodcastTitle: "Test Podcast",
			PodcastUUID:  "pod-1",
			URL:          "https://example.com/ep1.mp3",
			Published:    "2026-01-01T00:00:00Z",
			Duration:     3600,
			PlayedUpTo:   1800,
		},
		{
			UUID:         "ep-2",
			Title:        "Second Episode",
			PodcastTitle: "Test Podcast",
			PodcastUUID:  "pod-1",
			Duration:     1800,
			PlayedUpTo:   1800,
		},
	}

	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    int // expected episode count
		wantErr bool
	}{
		{
			name: "successful fetch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify Authorization header is present.
				if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
					t.Errorf("expected Bearer test-token, got %q", auth)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(HistoryResponse{
					Episodes: sampleEpisodes,
					Total:    2,
				})
			},
			want: 2,
		},
		{
			name: "unauthorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
			},
			wantErr: true,
		},
		{
			name: "empty history",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(HistoryResponse{
					Episodes: []Episode{},
					Total:    0,
				})
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			episodes, err := client.FetchHistory("test-token")
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchHistory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(episodes) != tt.want {
				t.Errorf("FetchHistory() returned %d episodes, want %d", len(episodes), tt.want)
			}
			// Verify status is populated on successful fetches with episodes.
			if !tt.wantErr && len(episodes) > 0 {
				for i, ep := range episodes {
					if ep.Status == "" {
						t.Errorf("episodes[%d].Status is empty, want derived value", i)
					}
				}
			}
		})
	}
}
