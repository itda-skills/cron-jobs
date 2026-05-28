package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
	if !strings.Contains(rec.Body.String(), "Run Test") {
		t.Fatalf("new job form missing Run Test")
	}
	if !strings.Contains(rec.Body.String(), "Script Content") {
		t.Fatalf("new job form missing script editor")
	}
}

func TestCreateJobRunTestShowsOutputWithoutSaving(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes())

	form := jobFormValues("draft", "Draft", "echo ui-test:$JOB_TEST_RUN\n")
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
	if _, ok := findJob(service.Config(), "draft"); ok {
		t.Fatalf("draft job was saved during test run")
	}
}

func TestCreateJobSavesScriptAndConfig(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes(httpapi.Server{Service: service}.Routes())

	form := jobFormValues("saved", "Saved", "echo saved\n")
	form.Set("action", "save")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	job, ok := findJob(service.Config(), "saved")
	if !ok {
		t.Fatalf("saved job not found")
	}
	content, err := service.ReadJobScript(job)
	if err != nil {
		t.Fatalf("ReadJobScript() error = %v", err)
	}
	if content != "echo saved\n" {
		t.Fatalf("script content = %q, want echo saved", content)
	}
}

func jobFormValues(id string, name string, script string) url.Values {
	form := url.Values{}
	form.Set("id", id)
	form.Set("name", name)
	form.Set("schedule_type", "daily")
	form.Set("time", "18:10")
	form.Set("timeout_seconds", "60")
	form.Set("script_content", script)
	form.Set("enabled", "on")
	return form
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
