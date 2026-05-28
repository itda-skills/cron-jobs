package scheduler

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/jobenv"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

func TestBuildPlanResolvesEnabledJobs(t *testing.T) {
	cfg, paths := testConfig(t)
	now := time.Date(2026, 5, 28, 17, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	planned, err := BuildPlan(cfg, paths, now, func(name string) (string, bool) {
		if name == "GITHUB_PAT" {
			return "secret", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(planned) != 1 {
		t.Fatalf("len(planned) = %d, want 1", len(planned))
	}
	if planned[0].Env["GITHUB_PAT"] != "secret" {
		t.Fatalf("planned env missing inherited secret")
	}
	if planned[0].NextRun.Format("15:04") != "18:10" {
		t.Fatalf("NextRun = %v, want 18:10", planned[0].NextRun)
	}
}

func TestBuildPlanReportsMissingInheritedEnv(t *testing.T) {
	cfg, paths := testConfig(t)
	_, err := BuildPlan(cfg, paths, time.Now(), func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("BuildPlan() error = nil, want missing inherited env error")
	}
}

func testConfig(t *testing.T) (config.Config, config.Paths) {
	t.Helper()
	dataDir := t.TempDir()
	paths := config.Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
		RecipeDir: filepath.Join(dataDir, "recipes"),
	}
	cfg := config.Config{
		Version:  1,
		Timezone: "Asia/Seoul",
		Env: config.EnvSection{
			Global: jobenv.Config{
				Plain:   map[string]string{"BRANCH": "main"},
				Inherit: []string{"GITHUB_PAT"},
			},
		},
		Recipes: []jobruntime.Recipe{{
			ID:       "github-actions",
			Language: jobruntime.LanguageBash,
			Path:     "recipes/bash/github-actions.sh",
		}},
		Jobs: []config.Job{{
			ID:      "weekday-report",
			Name:    "Weekday report",
			Enabled: true,
			Env: jobenv.Config{
				Plain: map[string]string{"OWNER": "itda-skills"},
			},
			Runtime: jobruntime.Config{
				Language:       jobruntime.LanguageBash,
				Script:         "scripts/jobs/weekday-report.sh",
				Recipes:        []string{"github-actions"},
				TimeoutSeconds: 60,
			},
			Schedule: schedule.Spec{
				Type:     schedule.TypeWeekly,
				Time:     "18:10",
				Weekdays: []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			},
		}, {
			ID:      "disabled",
			Name:    "Disabled",
			Enabled: false,
			Runtime: jobruntime.Config{
				Language:       jobruntime.LanguageBash,
				Script:         "scripts/jobs/disabled.sh",
				TimeoutSeconds: 60,
			},
			Schedule: schedule.Spec{Type: schedule.TypeDaily, Time: "12:00"},
		}},
	}
	return cfg, paths
}
