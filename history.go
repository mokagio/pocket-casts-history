package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HistoryOutput is the top-level JSON structure written to the output file.
type HistoryOutput struct {
	FetchedAt string    `json:"fetched_at"`
	Episodes  []Episode `json:"episodes"`
}

// NewHistoryOutput creates a HistoryOutput stamped with the current time.
//
// We accept a time.Time parameter instead of calling time.Now() internally.
// This makes the function deterministic and testable — a pattern sometimes
// called "dependency injection" for time.
func NewHistoryOutput(fetchedAt time.Time, episodes []Episode) HistoryOutput {
	return HistoryOutput{
		FetchedAt: fetchedAt.UTC().Format(time.RFC3339),
		Episodes:  episodes,
	}
}

// HistoryPath returns the path for the daily history file.
func HistoryPath(outputDir string, fetchedAt time.Time) string {
	date := fetchedAt.UTC()
	return filepath.Join(
		outputDir,
		fmt.Sprintf("%d", date.Year()),
		fmt.Sprintf("%02d", date.Month()),
		fmt.Sprintf("%02d.json", date.Day()),
	)
}

// LoadHistory reads and parses a history JSON file.
// Returns an empty HistoryOutput (not an error) if the file doesn't exist,
// so first run works cleanly.
func LoadHistory(path string) (HistoryOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HistoryOutput{}, nil
		}
		return HistoryOutput{}, fmt.Errorf("reading history file: %w", err)
	}

	var output HistoryOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return HistoryOutput{}, fmt.Errorf("parsing history file: %w", err)
	}

	return output, nil
}

// DiffEpisodes returns episodes that are new or have changed PlayedUpTo
// compared to the master list.
func DiffEpisodes(master []Episode, fetched []Episode) []Episode {
	index := make(map[string]Episode, len(master))
	for _, ep := range master {
		index[ep.UUID] = ep
	}

	var diff []Episode
	for _, ep := range fetched {
		existing, found := index[ep.UUID]
		if !found || existing.PlayedUpTo != ep.PlayedUpTo {
			diff = append(diff, ep)
		}
	}

	return diff
}

// writeJSON marshals v as indented JSON and writes it to path,
// creating parent directories as needed.
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// WriteHistory writes the history output to a JSON file.
//
// It creates the output directory if it doesn't exist, then writes the file
// with pretty-printed JSON (indented for human readability).
func WriteHistory(outputDir string, output HistoryOutput) error {
	path := filepath.Join(outputDir, "history.json")
	return writeJSON(path, output)
}

// UpdateHistory loads the master history, diffs the fetched episodes against
// it, writes a daily file with only new/changed episodes, and updates the
// master.
func UpdateHistory(outputDir string, fetchedAt time.Time, episodes []Episode) (int, error) {
	masterPath := filepath.Join(outputDir, "history.json")

	master, err := LoadHistory(masterPath)
	if err != nil {
		return 0, fmt.Errorf("loading master history: %w", err)
	}

	diff := DiffEpisodes(master.Episodes, episodes)

	// Write daily file with only new/changed episodes.
	dailyPath := HistoryPath(outputDir, fetchedAt)
	dailyOutput := NewHistoryOutput(fetchedAt, diff)
	if err := writeJSON(dailyPath, dailyOutput); err != nil {
		return 0, fmt.Errorf("writing daily history: %w", err)
	}

	// Merge fetched episodes into master, updating existing entries by UUID.
	merged := mergeEpisodes(master.Episodes, episodes)
	masterOutput := NewHistoryOutput(fetchedAt, merged)
	if err := writeJSON(masterPath, masterOutput); err != nil {
		return 0, fmt.Errorf("writing master history: %w", err)
	}

	return len(diff), nil
}

// mergeEpisodes merges fetched episodes into the master list.
// Existing episodes are updated; new ones are appended.
func mergeEpisodes(master []Episode, fetched []Episode) []Episode {
	index := make(map[string]int, len(master))
	merged := make([]Episode, len(master))
	copy(merged, master)

	for i, ep := range merged {
		index[ep.UUID] = i
	}

	for _, ep := range fetched {
		if i, found := index[ep.UUID]; found {
			merged[i] = ep
		} else {
			merged = append(merged, ep)
			index[ep.UUID] = len(merged) - 1
		}
	}

	return merged
}
