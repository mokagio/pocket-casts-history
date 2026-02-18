package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewHistoryOutput verifies the output struct is created correctly.
func TestNewHistoryOutput(t *testing.T) {
	// time.Date creates a deterministic timestamp for testing.
	ts := time.Date(2026, 2, 17, 12, 0, 0, 0, time.UTC)
	episodes := []Episode{
		{UUID: "ep-1", Title: "Test Episode"},
	}

	output := NewHistoryOutput(ts, episodes)

	if output.FetchedAt != "2026-02-17T12:00:00Z" {
		t.Errorf("FetchedAt = %q, want %q", output.FetchedAt, "2026-02-17T12:00:00Z")
	}
	if len(output.Episodes) != 1 {
		t.Errorf("Episodes count = %d, want 1", len(output.Episodes))
	}
}

// TestNewHistoryOutputConvertsToUTC ensures non-UTC times are normalized.
func TestNewHistoryOutputConvertsToUTC(t *testing.T) {
	// Create a time in a non-UTC timezone.
	loc := time.FixedZone("UTC+5", 5*60*60)
	ts := time.Date(2026, 2, 17, 17, 0, 0, 0, loc) // 17:00 UTC+5 = 12:00 UTC

	output := NewHistoryOutput(ts, nil)

	if output.FetchedAt != "2026-02-17T12:00:00Z" {
		t.Errorf("FetchedAt = %q, want %q (UTC-normalized)", output.FetchedAt, "2026-02-17T12:00:00Z")
	}
}

// TestWriteHistory verifies that the JSON file is written correctly.
func TestWriteHistory(t *testing.T) {
	dir := t.TempDir()

	output := HistoryOutput{
		FetchedAt: "2026-02-17T12:00:00Z",
		Episodes: []Episode{
			{
				UUID:         "ep-1",
				Title:        "Test Episode",
				PodcastTitle: "Test Podcast",
				PodcastUUID:  "pod-1",
				Duration:     3600,
				PlayedUpTo:   1800,
			},
		},
	}

	if err := WriteHistory(dir, output); err != nil {
		t.Fatalf("WriteHistory: %v", err)
	}

	// Read back and verify.
	path := filepath.Join(dir, "history.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var loaded HistoryOutput
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("parsing output file: %v", err)
	}

	if loaded.FetchedAt != output.FetchedAt {
		t.Errorf("FetchedAt = %q, want %q", loaded.FetchedAt, output.FetchedAt)
	}
	if len(loaded.Episodes) != 1 {
		t.Fatalf("Episodes count = %d, want 1", len(loaded.Episodes))
	}
	if loaded.Episodes[0].Title != "Test Episode" {
		t.Errorf("Episode title = %q, want %q", loaded.Episodes[0].Title, "Test Episode")
	}
}

// TestWriteHistoryCreatesDirectory checks that missing directories are created.
func TestWriteHistoryCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "output")

	output := HistoryOutput{
		FetchedAt: "2026-02-17T12:00:00Z",
		Episodes:  []Episode{},
	}

	if err := WriteHistory(dir, output); err != nil {
		t.Fatalf("WriteHistory: %v", err)
	}

	path := filepath.Join(dir, "history.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

// TestWriteHistoryEmptyEpisodes verifies output with no episodes.
func TestWriteHistoryEmptyEpisodes(t *testing.T) {
	dir := t.TempDir()

	output := HistoryOutput{
		FetchedAt: "2026-02-17T12:00:00Z",
		Episodes:  []Episode{},
	}

	if err := WriteHistory(dir, output); err != nil {
		t.Fatalf("WriteHistory: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "history.json"))
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var loaded HistoryOutput
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("parsing output file: %v", err)
	}

	if len(loaded.Episodes) != 0 {
		t.Errorf("Episodes count = %d, want 0", len(loaded.Episodes))
	}
}
