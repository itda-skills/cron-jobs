package scriptstore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWriteJobAndReadConfigured(t *testing.T) {
	dataDir := t.TempDir()
	store := Store{DataDir: dataDir, ScriptDir: filepath.Join(dataDir, "scripts", "jobs")}

	rel, err := store.WriteJob("weekday-report", "echo hello\n")
	if err != nil {
		t.Fatalf("WriteJob() error = %v", err)
	}
	if rel != "scripts/jobs/weekday-report.sh" {
		t.Fatalf("rel = %q, want scripts/jobs/weekday-report.sh", rel)
	}
	content, err := store.ReadConfigured(rel)
	if err != nil {
		t.Fatalf("ReadConfigured() error = %v", err)
	}
	if content != "echo hello\n" {
		t.Fatalf("content = %q, want echo hello", content)
	}
}

func TestWriteTestCreatesTemporaryScript(t *testing.T) {
	dataDir := t.TempDir()
	store := Store{DataDir: dataDir, ScriptDir: filepath.Join(dataDir, "scripts", "jobs")}
	rel, cleanup, err := store.WriteTest("draft", "echo test\n", time.Date(2026, 5, 28, 18, 10, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("WriteTest() error = %v", err)
	}
	defer cleanup()
	if rel == "" {
		t.Fatal("rel is empty")
	}
	content, err := store.ReadConfigured(rel)
	if err != nil {
		t.Fatalf("ReadConfigured() error = %v", err)
	}
	if content != "echo test\n" {
		t.Fatalf("content = %q, want echo test", content)
	}
}

func TestWriteTestNormalizesWindowsLineEndings(t *testing.T) {
	dataDir := t.TempDir()
	store := Store{DataDir: dataDir, ScriptDir: filepath.Join(dataDir, "scripts", "jobs")}
	rel, cleanup, err := store.WriteTest("draft", "#!/usr/bin/env bash\r\nset -euo pipefail\r\necho test\r\n", time.Date(2026, 5, 28, 18, 10, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("WriteTest() error = %v", err)
	}
	defer cleanup()
	content, err := store.ReadConfigured(rel)
	if err != nil {
		t.Fatalf("ReadConfigured() error = %v", err)
	}
	want := "#!/usr/bin/env bash\nset -euo pipefail\necho test\n"
	if content != want {
		t.Fatalf("content = %q, want %q", content, want)
	}
}

func TestWriteJobRejectsUnsafeID(t *testing.T) {
	store := Store{DataDir: t.TempDir(), ScriptDir: t.TempDir()}
	if _, err := store.WriteJob("../bad", "echo bad\n"); err == nil {
		t.Fatal("WriteJob() error = nil for unsafe id")
	}
}
