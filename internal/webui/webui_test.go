package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
	if strings.Contains(rec.Body.String(), "Raw Config") {
		t.Fatalf("dashboard exposes raw config editor")
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
	if !strings.Contains(rec.Body.String(), "Run Test") {
		t.Fatalf("new job form missing Run Test")
	}
	if !strings.Contains(rec.Body.String(), "Script Content") {
		t.Fatalf("new job form missing script editor")
	}
	if strings.Contains(rec.Body.String(), "<label>ID</label>") {
		t.Fatalf("new job form exposes internal ID field")
	}
	if strings.Contains(rec.Body.String(), "Saved Script Path") {
		t.Fatalf("new job form exposes internal script path field")
	}
}

func TestCreateJobRunTestShowsOutputWithoutSaving(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes())

	form := jobFormValues("Draft", "#!/usr/bin/env bash\r\nset -euo pipefail\r\necho ui-test:$JOB_TEST_RUN\r\n")
	form.Set("action", "test")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ui-test:true") {
		t.Fatalf("test output not rendered: %s", body)
	}
	if _, ok := findJobByName(service.Config(), "Draft"); ok {
		t.Fatalf("draft job was saved during test run")
	}
}

func TestCreateJobSavesScriptAndConfig(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes())

	form := jobFormValues("Saved", "echo saved\n")
	form.Set("action", "save")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	job, ok := findJobByName(service.Config(), "Saved")
	if !ok {
		t.Fatalf("saved job not found")
	}
	if !uuid7Pattern.MatchString(job.ID) {
		t.Fatalf("saved job id = %q, want generated UUIDv7", job.ID)
	}
	content, err := service.ReadJobScript(job)
	if err != nil {
		t.Fatalf("ReadJobScript() error = %v", err)
	}
	if content != "echo saved\n" {
		t.Fatalf("script content = %q, want echo saved", content)
	}
}

var uuid7Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func jobFormValues(name string, script string) url.Values {
	form := url.Values{}
	form.Set("name", name)
	form.Set("schedule_type", "daily")
	form.Set("time", "18:10")
	form.Set("timeout_seconds", "60")
	form.Set("script_content", script)
	form.Set("enabled", "on")
	return form
}

func findJobByName(cfg config.Config, name string) (config.Job, bool) {
	for _, job := range cfg.Jobs {
		if job.Name == name {
			return job, true
		}
	}
	return config.Job{}, false
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
