package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

func TestRunDueExecutesJobAndAdvancesNextRun(t *testing.T) {
	dataDir := t.TempDir()
	scriptDir := filepath.Join(dataDir, "scripts", "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "daily.sh"), []byte("echo due-run\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	settings := Settings{
		Addr:       ":0",
		DataDir:    dataDir,
		ConfigPath: filepath.Join(dataDir, "config.json"),
		LogDir:     filepath.Join(dataDir, "logs"),
		ScriptDir:  scriptDir,
		Timezone:   "Asia/Seoul",
	}
	service := NewService(settings)
	cfg := config.Config{
		Version:  1,
		Timezone: "Asia/Seoul",
		Jobs: []config.Job{{
			ID:      "daily",
			Name:    "Daily",
			Enabled: true,
			Runtime: jobruntime.Config{
				Language:       jobruntime.LanguageBash,
				Script:         "scripts/jobs/daily.sh",
				TimeoutSeconds: 5,
			},
			Schedule: schedule.Spec{Type: schedule.TypeDaily, Time: "18:10"},
		}},
	}

	loc := time.FixedZone("KST", 9*60*60)
	if err := service.applyConfig(cfg, time.Date(2026, 5, 28, 17, 0, 0, 0, loc)); err != nil {
		t.Fatalf("applyConfig() error = %v", err)
	}
	service.RunDue(context.Background(), time.Date(2026, 5, 28, 18, 10, 1, 0, loc))

	var runsCount int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runs, err := service.RecentRuns(10)
		if err != nil {
			t.Fatalf("RecentRuns() error = %v", err)
		}
		runsCount = len(runs)
		if runsCount == 1 {
			if runs[0].Status != "success" {
				t.Fatalf("run status = %q, want success", runs[0].Status)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if runsCount != 1 {
		t.Fatalf("runsCount = %d, want 1", runsCount)
	}

	jobs := service.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	if !jobs[0].NextRun.After(time.Date(2026, 5, 28, 18, 10, 1, 0, loc)) {
		t.Fatalf("next run was not advanced: %v", jobs[0].NextRun)
	}
}

func TestTestJobRunsDraftScriptWithoutSavingConfig(t *testing.T) {
	dataDir := t.TempDir()
	scriptDir := filepath.Join(dataDir, "scripts", "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := Settings{
		Addr:       ":0",
		DataDir:    dataDir,
		ConfigPath: filepath.Join(dataDir, "config.json"),
		LogDir:     filepath.Join(dataDir, "logs"),
		ScriptDir:  scriptDir,
		Timezone:   "Asia/Seoul",
	}
	service := NewService(settings)
	if err := service.applyConfig(config.Config{Version: 1, Timezone: "Asia/Seoul"}, time.Now()); err != nil {
		t.Fatalf("applyConfig() error = %v", err)
	}

	job := config.Job{
		ID:      "draft",
		Name:    "Draft",
		Enabled: true,
		Runtime: jobruntime.Config{
			Language:       jobruntime.LanguageBash,
			TimeoutSeconds: 5,
		},
		Schedule: schedule.Spec{Type: schedule.TypeDaily, Time: "18:10"},
	}
	entry, output, err := service.TestJob(context.Background(), job, "echo reason:$JOB_RUN_REASON\necho test:$JOB_TEST_RUN\n")
	if err != nil {
		t.Fatalf("TestJob() error = %v, output = %s", err, output)
	}
	if entry.RunReason != "test" {
		t.Fatalf("RunReason = %q, want test", entry.RunReason)
	}
	if entry.Status != "success" {
		t.Fatalf("Status = %q, want success", entry.Status)
	}
	if output != "reason:test\ntest:true\n" {
		t.Fatalf("output = %q, want test env output", output)
	}
	if _, err := os.Stat(filepath.Join(scriptDir, "draft.sh")); !os.IsNotExist(err) {
		t.Fatalf("saved job script exists after test run, err = %v", err)
	}
}
