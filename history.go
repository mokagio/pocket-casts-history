package main

import (
	"encoding/json"
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

// WriteHistory writes the history output to a JSON file.
//
// It creates the output directory if it doesn't exist, then writes the file
// with pretty-printed JSON (indented for human readability).
func WriteHistory(outputDir string, output HistoryOutput) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// json.MarshalIndent produces human-readable JSON with the given prefix
	// and indent strings. Empty prefix + two-space indent is conventional.
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling history: %w", err)
	}

	// Append a newline — many tools expect text files to end with one.
	data = append(data, '\n')

	path := filepath.Join(outputDir, "history.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing history file: %w", err)
	}

	return nil
}
