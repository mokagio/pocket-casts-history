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

func TestLoadHistory(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "history.json")

		output := HistoryOutput{
			FetchedAt: "2026-02-17T12:00:00Z",
			Episodes: []Episode{
				{UUID: "ep-1", Title: "Existing"},
			},
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		os.WriteFile(path, data, 0o644)

		loaded, err := LoadHistory(path)
		if err != nil {
			t.Fatalf("LoadHistory: %v", err)
		}
		if len(loaded.Episodes) != 1 {
			t.Fatalf("Episodes count = %d, want 1", len(loaded.Episodes))
		}
		if loaded.Episodes[0].UUID != "ep-1" {
			t.Errorf("UUID = %q, want %q", loaded.Episodes[0].UUID, "ep-1")
		}
	})

	t.Run("file missing", func(t *testing.T) {
		loaded, err := LoadHistory("/nonexistent/history.json")
		if err != nil {
			t.Fatalf("LoadHistory: %v", err)
		}
		if len(loaded.Episodes) != 0 {
			t.Errorf("Episodes count = %d, want 0", len(loaded.Episodes))
		}
	})

	t.Run("corrupt file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "history.json")
		os.WriteFile(path, []byte("not json"), 0o644)

		_, err := LoadHistory(path)
		if err == nil {
			t.Fatal("expected error for corrupt file, got nil")
		}
	})
}

func TestDiffEpisodes(t *testing.T) {
	t.Run("empty master", func(t *testing.T) {
		fetched := []Episode{
			{UUID: "ep-1", Title: "New", PlayedUpTo: 100},
			{UUID: "ep-2", Title: "Also New", PlayedUpTo: 200},
		}

		diff := DiffEpisodes(nil, fetched)
		if len(diff) != 2 {
			t.Fatalf("diff count = %d, want 2", len(diff))
		}
	})

	t.Run("all unchanged", func(t *testing.T) {
		master := []Episode{
			{UUID: "ep-1", PlayedUpTo: 100},
			{UUID: "ep-2", PlayedUpTo: 200},
		}
		fetched := []Episode{
			{UUID: "ep-1", PlayedUpTo: 100},
			{UUID: "ep-2", PlayedUpTo: 200},
		}

		diff := DiffEpisodes(master, fetched)
		if len(diff) != 0 {
			t.Fatalf("diff count = %d, want 0", len(diff))
		}
	})

	t.Run("mix of new unchanged and progressed", func(t *testing.T) {
		master := []Episode{
			{UUID: "ep-1", PlayedUpTo: 100},
			{UUID: "ep-2", PlayedUpTo: 200},
		}
		fetched := []Episode{
			{UUID: "ep-1", PlayedUpTo: 100}, // unchanged
			{UUID: "ep-2", PlayedUpTo: 500}, // progressed
			{UUID: "ep-3", PlayedUpTo: 50},  // new
		}

		diff := DiffEpisodes(master, fetched)
		if len(diff) != 2 {
			t.Fatalf("diff count = %d, want 2", len(diff))
		}

		uuids := map[string]bool{}
		for _, ep := range diff {
			uuids[ep.UUID] = true
		}
		if !uuids["ep-2"] {
			t.Error("expected ep-2 (progressed) in diff")
		}
		if !uuids["ep-3"] {
			t.Error("expected ep-3 (new) in diff")
		}
	})

	t.Run("episode finished", func(t *testing.T) {
		master := []Episode{
			{UUID: "ep-1", PlayedUpTo: 1000, Duration: 3600},
		}
		fetched := []Episode{
			{UUID: "ep-1", PlayedUpTo: 3600, Duration: 3600},
		}

		diff := DiffEpisodes(master, fetched)
		if len(diff) != 1 {
			t.Fatalf("diff count = %d, want 1", len(diff))
		}
		if diff[0].PlayedUpTo != 3600 {
			t.Errorf("PlayedUpTo = %d, want 3600", diff[0].PlayedUpTo)
		}
	})
}

func TestUpdateHistory(t *testing.T) {
	t.Run("first run", func(t *testing.T) {
		dir := t.TempDir()
		ts := time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC)
		episodes := []Episode{
			{UUID: "ep-1", Title: "First", PlayedUpTo: 100},
			{UUID: "ep-2", Title: "Second", PlayedUpTo: 200},
		}

		count, err := UpdateHistory(dir, ts, episodes)
		if err != nil {
			t.Fatalf("UpdateHistory: %v", err)
		}
		if count != 2 {
			t.Errorf("new count = %d, want 2", count)
		}

		// Daily file should have all episodes.
		dailyPath := filepath.Join(dir, "2026", "02", "19.json")
		daily, err := LoadHistory(dailyPath)
		if err != nil {
			t.Fatalf("loading daily: %v", err)
		}
		if len(daily.Episodes) != 2 {
			t.Errorf("daily episodes = %d, want 2", len(daily.Episodes))
		}

		// Master should have all episodes.
		master, err := LoadHistory(filepath.Join(dir, "history.json"))
		if err != nil {
			t.Fatalf("loading master: %v", err)
		}
		if len(master.Episodes) != 2 {
			t.Errorf("master episodes = %d, want 2", len(master.Episodes))
		}
	})

	t.Run("second run with changes", func(t *testing.T) {
		dir := t.TempDir()
		ts1 := time.Date(2026, 2, 18, 12, 0, 0, 0, time.UTC)
		ts2 := time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC)

		// First run seeds the master.
		_, err := UpdateHistory(dir, ts1, []Episode{
			{UUID: "ep-1", Title: "First", PlayedUpTo: 100},
			{UUID: "ep-2", Title: "Second", PlayedUpTo: 200},
		})
		if err != nil {
			t.Fatalf("first UpdateHistory: %v", err)
		}

		// Second run: ep-2 progressed, ep-3 is new.
		count, err := UpdateHistory(dir, ts2, []Episode{
			{UUID: "ep-1", Title: "First", PlayedUpTo: 100},  // unchanged
			{UUID: "ep-2", Title: "Second", PlayedUpTo: 500}, // progressed
			{UUID: "ep-3", Title: "Third", PlayedUpTo: 50},   // new
		})
		if err != nil {
			t.Fatalf("second UpdateHistory: %v", err)
		}
		if count != 2 {
			t.Errorf("new count = %d, want 2", count)
		}

		// Daily file should have only the delta.
		dailyPath := filepath.Join(dir, "2026", "02", "19.json")
		daily, err := LoadHistory(dailyPath)
		if err != nil {
			t.Fatalf("loading daily: %v", err)
		}
		if len(daily.Episodes) != 2 {
			t.Errorf("daily episodes = %d, want 2", len(daily.Episodes))
		}

		// Master should have all 3 episodes with updated values.
		master, err := LoadHistory(filepath.Join(dir, "history.json"))
		if err != nil {
			t.Fatalf("loading master: %v", err)
		}
		if len(master.Episodes) != 3 {
			t.Errorf("master episodes = %d, want 3", len(master.Episodes))
		}
	})

	t.Run("no changes", func(t *testing.T) {
		dir := t.TempDir()
		ts1 := time.Date(2026, 2, 18, 12, 0, 0, 0, time.UTC)
		ts2 := time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC)
		episodes := []Episode{
			{UUID: "ep-1", Title: "First", PlayedUpTo: 100},
		}

		_, err := UpdateHistory(dir, ts1, episodes)
		if err != nil {
			t.Fatalf("first UpdateHistory: %v", err)
		}

		count, err := UpdateHistory(dir, ts2, episodes)
		if err != nil {
			t.Fatalf("second UpdateHistory: %v", err)
		}
		if count != 0 {
			t.Errorf("new count = %d, want 0", count)
		}

		// Daily file should have zero episodes.
		dailyPath := filepath.Join(dir, "2026", "02", "19.json")
		daily, err := LoadHistory(dailyPath)
		if err != nil {
			t.Fatalf("loading daily: %v", err)
		}
		if len(daily.Episodes) != 0 {
			t.Errorf("daily episodes = %d, want 0", len(daily.Episodes))
		}
	})
}
