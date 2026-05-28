package logstore

import (
	"testing"
	"time"
)

func TestAppendAndRecent(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	now := time.Date(2026, 5, 28, 18, 10, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		entry := Entry{
			RunID:       NewRunID("job", now.Add(time.Duration(i)*time.Minute)),
			JobID:       "job",
			ScheduledAt: now.Add(time.Duration(i) * time.Minute),
			Status:      StatusSuccess,
			LogPath:     "runs/2026-05-28/job.log",
		}
		if err := store.Append(entry); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := store.Recent(2)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Recent()) = %d, want 2", len(got))
	}
	if !got[0].ScheduledAt.After(got[1].ScheduledAt) {
		t.Fatalf("Recent() = %#v, want newest first", got)
	}
}

func TestCreateRunLogAndRead(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	_, rel, file, err := store.CreateRunLog("job", time.Date(2026, 5, 28, 18, 10, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateRunLog() error = %v", err)
	}
	if _, err := file.WriteString("hello\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := store.ReadLog(Entry{LogPath: rel})
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("ReadLog() = %q, want hello", data)
	}
}
