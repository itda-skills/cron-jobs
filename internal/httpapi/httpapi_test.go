package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/itda-skills/cron-jobs/internal/app"
	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

func TestAPIListJobs(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var jobs []app.JobView
	if err := json.Unmarshal(rec.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("json decode error = %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "daily" {
		t.Fatalf("jobs = %#v, want daily", jobs)
	}
}

func TestAPIRunJobAndReadLog(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/daily/run", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("run status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var entry struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &entry); err != nil {
		t.Fatalf("json decode error = %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/runs/"+entry.RunID+"/log", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("log status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "hello api\n" {
		t.Fatalf("log body = %q, want hello api", rec.Body.String())
	}
}

func TestAPITestJobReturnsOutput(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes()

	body := bytes.NewBufferString(`{"job":{"id":"draft","name":"Draft","enabled":true,"runtime":{"language":"bash","script":"","timeout_seconds":60},"schedule":{"type":"daily","time":"18:10"}},"script_content":"echo api-test:$JOB_TEST_RUN\n"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/test", body)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode error = %v", err)
	}
	if resp.Output != "api-test:true\n" {
		t.Fatalf("output = %q, want api-test:true", resp.Output)
	}
}

func TestAPIPutConfigRejectsInvalidConfig(t *testing.T) {
	service := testService(t)
	handler := Server{Service: service}.Routes()

	body := bytes.NewBufferString(`{"version":1,"timezone":"Asia/Seoul","jobs":[{"id":"bad","name":"Bad","enabled":true,"runtime":{"language":"bash","script":"../bad.sh","timeout_seconds":60},"schedule":{"type":"daily","time":"12:00"}}]}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", body)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func testService(t *testing.T) *app.Service {
	t.Helper()
	dataDir := t.TempDir()
	scriptDir := filepath.Join(dataDir, "scripts", "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "daily.sh"), []byte("echo hello api\n"), 0o644); err != nil {
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
