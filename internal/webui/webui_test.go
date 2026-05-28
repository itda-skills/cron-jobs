package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itda-skills/cron-jobs/internal/app"
	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/httpapi"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

func TestDashboardRendersJobs(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Daily") {
		t.Fatalf("dashboard does not contain job name: %s", rec.Body.String())
	}
}

func TestNewJobFormRenders(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs/new", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Weekdays") {
		t.Fatalf("new job form missing weekdays")
	}
}

func testService(t *testing.T) *app.Service {
	t.Helper()
	dataDir := t.TempDir()
	scriptDir := filepath.Join(dataDir, "scripts", "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "daily.sh"), []byte("echo hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	settings := app.Settings{
		Addr:       ":0",
		DataDir:    dataDir,
		ConfigPath: filepath.Join(dataDir, "config.json"),
		LogDir:     filepath.Join(dataDir, "logs"),
		ScriptDir:  scriptDir,
		RecipeDir:  filepath.Join(dataDir, "recipes"),
		Timezone:   "Asia/Seoul",
	}
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
				TimeoutSeconds: 60,
			},
			Schedule: schedule.Spec{Type: schedule.TypeDaily, Time: "18:10"},
		}},
	}
	if err := config.Save(settings.ConfigPath, cfg); err != nil {
		t.Fatal(err)
	}
	service := app.NewService(settings)
	if err := service.Load(); err != nil {
		t.Fatal(err)
	}
	return service
}
