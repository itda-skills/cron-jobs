package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itda-skills/cron-jobs/internal/jobenv"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg, paths := validConfig(t)
	if err := cfg.Validate(paths); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsDuplicateJobID(t *testing.T) {
	cfg, paths := validConfig(t)
	cfg.Jobs = append(cfg.Jobs, cfg.Jobs[0])
	if err := cfg.Validate(paths); err == nil {
		t.Fatal("Validate() error = nil for duplicate job id")
	}
}

func TestValidateRejectsInvalidSchedule(t *testing.T) {
	cfg, paths := validConfig(t)
	cfg.Jobs[0].Schedule.Time = "8:10"
	if err := cfg.Validate(paths); err == nil {
		t.Fatal("Validate() error = nil for invalid schedule")
	}
}

func TestValidateRejectsPathTraversal(t *testing.T) {
	cfg, paths := validConfig(t)
	cfg.Jobs[0].Runtime.Script = "../outside.sh"
	err := cfg.Validate(paths)
	if err == nil || !strings.Contains(err.Error(), "script") {
		t.Fatalf("Validate() error = %v, want script path error", err)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	cfg, _ := validConfig(t)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved config stat error = %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Jobs[0].ID != cfg.Jobs[0].ID {
		t.Fatalf("loaded job id = %q, want %q", got.Jobs[0].ID, cfg.Jobs[0].ID)
	}
}

func validConfig(t *testing.T) (Config, Paths) {
	t.Helper()
	dataDir := t.TempDir()
	paths := Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
		RecipeDir: filepath.Join(dataDir, "recipes"),
	}
	cfg := Config{
		Version:  1,
		Timezone: "Asia/Seoul",
		Env: EnvSection{
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
		Jobs: []Job{{
			ID:      "weekday_report",
			Name:    "Weekday report",
			Enabled: true,
			Env: jobenv.Config{
				Plain: map[string]string{
					"OWNER":         "itda-skills",
					"REPO":          "rs-golden-queens",
					"WORKFLOW_FILE": "flow-kr.yml",
				},
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
		}},
	}
	return cfg, paths
}
