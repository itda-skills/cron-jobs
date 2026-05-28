package logstore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	StatusSuccess  = "success"
	StatusFailed   = "failed"
	StatusTimedOut = "timed_out"
	StatusSkipped  = "skipped"
)

var safeNamePattern = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

type Store struct {
	Dir string
}

type Entry struct {
	RunID          string    `json:"run_id"`
	JobID          string    `json:"job_id"`
	JobName        string    `json:"job_name"`
	RunReason      string    `json:"run_reason,omitempty"`
	ScheduledAt    time.Time `json:"scheduled_at"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	DurationMillis int64     `json:"duration_millis"`
	ExitCode       int       `json:"exit_code"`
	Status         string    `json:"status"`
	LogPath        string    `json:"log_path"`
	Error          string    `json:"error,omitempty"`
}

func (s Store) Ensure() error {
	if s.Dir == "" {
		return fmt.Errorf("log dir is required")
	}
	if err := os.MkdirAll(filepath.Join(s.Dir, "runs"), 0o755); err != nil {
		return err
	}
	return nil
}

func (s Store) CreateRunLog(jobID string, scheduledAt time.Time) (string, string, *os.File, error) {
	if err := s.Ensure(); err != nil {
		return "", "", nil, err
	}
	runID := NewRunID(jobID, scheduledAt)
	day := scheduledAt.Format("2006-01-02")
	relPath := filepath.Join("runs", day, runID+".log")
	absPath := filepath.Join(s.Dir, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", "", nil, err
	}
	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", "", nil, err
	}
	return runID, relPath, file, nil
}

func (s Store) Append(entry Entry) error {
	if err := s.Ensure(); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, "index.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(entry)
}

func (s Store) Recent(limit int) ([]Entry, error) {
	path := filepath.Join(s.Dir, "index.jsonl")
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit >= len(entries) {
		reverse(entries)
		return entries, nil
	}
	entries = entries[len(entries)-limit:]
	reverse(entries)
	return entries, nil
}

func (s Store) ReadLog(entry Entry) ([]byte, error) {
	if entry.LogPath == "" {
		return nil, fmt.Errorf("log path is empty")
	}
	joined := filepath.Join(s.Dir, entry.LogPath)
	cleanDir, err := filepath.Abs(s.Dir)
	if err != nil {
		return nil, err
	}
	cleanPath, err := filepath.Abs(joined)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil {
		return nil, err
	}
	if rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return nil, fmt.Errorf("log path escapes log dir")
	}
	return os.ReadFile(cleanPath)
}

func NewRunID(jobID string, scheduledAt time.Time) string {
	safeJobID := safeNamePattern.ReplaceAllString(jobID, "_")
	return scheduledAt.UTC().Format("20060102T150405.000000000Z") + "-" + safeJobID
}

func reverse(entries []Entry) {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
}
