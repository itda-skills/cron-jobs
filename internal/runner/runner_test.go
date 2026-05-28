package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/logstore"
)

func TestRunBashJobWithConstrainedEnv(t *testing.T) {
	dataDir := t.TempDir()
	scriptDir := filepath.Join(dataDir, "scripts", "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(scriptDir, "job.sh")
	if err := os.WriteFile(scriptPath, []byte("echo job:$JOB_ID\necho secret:$GITHUB_PAT\necho reason:$JOB_RUN_REASON\necho test:$JOB_TEST_RUN\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := logstore.Store{Dir: filepath.Join(dataDir, "logs")}
	runner := Runner{Store: store, ConfigPath: filepath.Join(dataDir, "config.json"), WorkDirRoot: dataDir}
	entry, err := runner.Run(context.Background(), Job{
		ID:           "weekday-report",
		Name:         "Weekday report",
		ScheduleType: "weekly",
		ScheduledAt:  time.Date(2026, 5, 28, 18, 10, 0, 0, time.UTC),
		RunReason:    RunReasonManual,
		Runtime: jobruntime.Resolved{
			Language: jobruntime.LanguageBash,
			Script:   scriptPath,
			Timeout:  time.Second,
		},
		Env: map[string]string{
			"GITHUB_PAT": "test-token",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if entry.Status != logstore.StatusSuccess {
		t.Fatalf("Status = %q, want success", entry.Status)
	}
	data, err := store.ReadLog(entry)
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	log := string(data)
	for _, want := range []string{"job:weekday-report", "secret:test-token", "reason:manual", "test:false"} {
		if !strings.Contains(log, want) {
			t.Fatalf("log %q does not contain %q", log, want)
		}
	}
}

func TestRunTimesOut(t *testing.T) {
	dataDir := t.TempDir()
	scriptDir := filepath.Join(dataDir, "scripts", "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(scriptDir, "sleep.sh")
	if err := os.WriteFile(scriptPath, []byte("sleep 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := Runner{Store: logstore.Store{Dir: filepath.Join(dataDir, "logs")}, WorkDirRoot: dataDir}
	entry, err := runner.Run(context.Background(), Job{
		ID:          "slow",
		Name:        "Slow",
		ScheduledAt: time.Date(2026, 5, 28, 18, 10, 0, 0, time.UTC),
		RunReason:   RunReasonManual,
		Runtime: jobruntime.Resolved{
			Language: jobruntime.LanguageBash,
			Script:   scriptPath,
			Timeout:  50 * time.Millisecond,
		},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want timeout")
	}
	if entry.Status != logstore.StatusTimedOut {
		t.Fatalf("Status = %q, want timed_out", entry.Status)
	}
}
