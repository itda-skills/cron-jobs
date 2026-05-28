package webui

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/itda-skills/cron-jobs/internal/app"
	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/idgen"
	"github.com/itda-skills/cron-jobs/internal/jobenv"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/logstore"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

type Server struct {
	Service *app.Service
}

type pageData struct {
	Title       string
	Config      config.Config
	Jobs        []app.JobView
	Runs        []logstore.Entry
	Run         logstore.Entry
	RunLog      string
	RunID       string
	FormJob     config.Job
	FormAction  string
	ScriptText  string
	TestOutput  string
	TestStatus  string
	Error       string
	WeekdayList []weekdayOption
}

type weekdayOption struct {
	Value   string
	Label   string
	Checked bool
}

func (s Server) Routes(api http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.dashboard)
	mux.HandleFunc("GET /jobs/new", s.newJob)
	mux.HandleFunc("POST /jobs", s.createJob)
	mux.HandleFunc("GET /jobs/{id}/edit", s.editJob)
	mux.HandleFunc("POST /jobs/{id}", s.updateJob)
	mux.HandleFunc("POST /jobs/{id}/toggle", s.toggleJob)
	mux.HandleFunc("POST /jobs/{id}/run", s.runJob)
	mux.HandleFunc("GET /runs/{id}", s.runLog)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s Server) dashboard(w http.ResponseWriter, r *http.Request) {
	cfg := s.Service.Config()
	runs, err := s.Service.RecentRuns(20)
	if err != nil {
		s.render(w, http.StatusInternalServerError, "dashboard", s.data("Dashboard", cfg, err.Error()))
		return
	}
	data := s.data("Dashboard", cfg, "")
	data.Jobs = s.Service.ListJobs()
	data.Runs = runs
	s.render(w, http.StatusOK, "dashboard", data)
}

func (s Server) newJob(w http.ResponseWriter, r *http.Request) {
	job := config.Job{
		Enabled: true,
		Runtime: jobruntime.Config{
			Language:       jobruntime.LanguageBash,
			TimeoutSeconds: 60,
		},
		Schedule: schedule.Spec{Type: schedule.TypeDaily, Time: "18:10"},
	}
	data := s.data("New Job", s.Service.Config(), "")
	data.FormJob = job
	data.FormAction = "/jobs"
	data.ScriptText = defaultScriptText()
	s.render(w, http.StatusOK, "job_form", data)
}

func (s Server) createJob(w http.ResponseWriter, r *http.Request) {
	cfg := s.Service.Config()
	job, scriptText, err := parseJobForm(r, "")
	if err != nil {
		s.renderJobForm(w, http.StatusBadRequest, "New Job", cfg, job, scriptText, "/jobs", err, "", "")
		return
	}
	if r.FormValue("action") == "test" {
		s.renderTestJob(r.Context(), w, "New Job", cfg, job, scriptText, "/jobs")
		return
	}
	scriptPath, err := s.Service.SaveJobScript(job.ID, scriptText)
	if err != nil {
		s.renderJobForm(w, http.StatusBadRequest, "New Job", cfg, job, scriptText, "/jobs", err, "", "")
		return
	}
	job.Runtime.Script = scriptPath
	cfg.Jobs = append(cfg.Jobs, job)
	if err := s.Service.SaveConfig(cfg); err != nil {
		s.renderJobForm(w, http.StatusBadRequest, "New Job", cfg, job, scriptText, "/jobs", err, "", "")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s Server) editJob(w http.ResponseWriter, r *http.Request) {
	cfg := s.Service.Config()
	job, ok := findJob(cfg, r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := s.data("Edit Job", cfg, "")
	data.FormJob = job
	data.FormAction = "/jobs/" + job.ID
	scriptText, err := s.Service.ReadJobScript(job)
	if err != nil {
		data.Error = err.Error()
	}
	data.ScriptText = scriptText
	s.render(w, http.StatusOK, "job_form", data)
}

func (s Server) updateJob(w http.ResponseWriter, r *http.Request) {
	cfg := s.Service.Config()
	id := r.PathValue("id")
	job, scriptText, err := parseJobForm(r, id)
	if err != nil {
		s.renderJobForm(w, http.StatusBadRequest, "Edit Job", cfg, job, scriptText, "/jobs/"+id, err, "", "")
		return
	}
	if r.FormValue("action") == "test" {
		s.renderTestJob(r.Context(), w, "Edit Job", cfg, job, scriptText, "/jobs/"+id)
		return
	}
	scriptPath, err := s.Service.SaveJobScript(job.ID, scriptText)
	if err != nil {
		s.renderJobForm(w, http.StatusBadRequest, "Edit Job", cfg, job, scriptText, "/jobs/"+id, err, "", "")
		return
	}
	job.Runtime.Script = scriptPath
	updated := false
	for i := range cfg.Jobs {
		if cfg.Jobs[i].ID == id {
			cfg.Jobs[i] = job
			updated = true
			break
		}
	}
	if !updated {
		http.NotFound(w, r)
		return
	}
	if err := s.Service.SaveConfig(cfg); err != nil {
		s.renderJobForm(w, http.StatusBadRequest, "Edit Job", cfg, job, scriptText, "/jobs/"+id, err, "", "")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s Server) toggleJob(w http.ResponseWriter, r *http.Request) {
	cfg := s.Service.Config()
	id := r.PathValue("id")
	for i := range cfg.Jobs {
		if cfg.Jobs[i].ID == id {
			cfg.Jobs[i].Enabled = !cfg.Jobs[i].Enabled
			if err := s.Service.SaveConfig(cfg); err != nil {
				s.render(w, http.StatusBadRequest, "dashboard", s.data("Dashboard", cfg, err.Error()))
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	http.NotFound(w, r)
}

func (s Server) runJob(w http.ResponseWriter, r *http.Request) {
	if _, err := s.Service.RunJobNow(r.Context(), r.PathValue("id")); err != nil {
		s.render(w, http.StatusBadRequest, "dashboard", s.data("Dashboard", s.Service.Config(), err.Error()))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s Server) runLog(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	entry, err := s.Service.FindRun(runID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	logData, err := s.Service.ReadRunLog(runID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := s.data("Run Log", s.Service.Config(), "")
	data.Run = entry
	data.RunID = runID
	data.RunLog = string(logData)
	s.render(w, http.StatusOK, "run_log", data)
}

func (s Server) data(title string, cfg config.Config, errText string) pageData {
	return pageData{
		Title:  title,
		Config: cfg,
		Error:  errText,
	}
}

func (s Server) renderJobForm(w http.ResponseWriter, status int, title string, cfg config.Config, job config.Job, scriptText string, action string, err error, testOutput string, testStatus string) {
	data := s.data(title, cfg, err.Error())
	data.FormJob = job
	data.FormAction = action
	data.ScriptText = scriptText
	data.TestOutput = testOutput
	data.TestStatus = testStatus
	s.render(w, status, "job_form", data)
}

func (s Server) renderTestJob(ctx context.Context, w http.ResponseWriter, title string, cfg config.Config, job config.Job, scriptText string, action string) {
	entry, output, err := s.Service.TestJob(ctx, job, scriptText)
	status := "success"
	errText := ""
	if entry.Status != "" {
		status = entry.Status
	}
	if err != nil {
		if entry.RunID == "" {
			s.renderJobForm(w, http.StatusBadRequest, title, cfg, job, scriptText, action, err, output, status)
			return
		}
		errText = err.Error()
	}
	data := s.data(title, cfg, errText)
	data.FormJob = job
	data.FormAction = action
	data.ScriptText = scriptText
	data.TestOutput = output
	data.TestStatus = status
	s.render(w, http.StatusOK, "job_form", data)
}

func (s Server) render(w http.ResponseWriter, status int, name string, data pageData) {
	data.WeekdayList = weekdays(data.FormJob.Schedule.Weekdays)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = templates.ExecuteTemplate(w, name, data)
}

func parseJobForm(r *http.Request, existingID string) (config.Job, string, error) {
	if err := r.ParseForm(); err != nil {
		return config.Job{}, "", err
	}
	var id string
	if existingID != "" {
		id = existingID
	} else {
		generated, err := idgen.NewUUIDv7()
		if err != nil {
			return config.Job{}, r.FormValue("script_content"), err
		}
		id = generated
	}
	timeout, err := strconv.Atoi(strings.TrimSpace(r.FormValue("timeout_seconds")))
	if err != nil {
		return config.Job{}, r.FormValue("script_content"), fmt.Errorf("timeout_seconds must be a number")
	}
	job := config.Job{
		ID:      id,
		Name:    strings.TrimSpace(r.FormValue("name")),
		Enabled: r.FormValue("enabled") == "on",
		Env: jobenv.Config{
			Plain:   parsePlainEnv(r.FormValue("env_plain")),
			Inherit: splitCSV(r.FormValue("env_inherit")),
		},
		Runtime: jobruntime.Config{
			Language:       jobruntime.LanguageBash,
			TimeoutSeconds: timeout,
		},
		Schedule: schedule.Spec{
			Type: strings.TrimSpace(r.FormValue("schedule_type")),
			Time: strings.TrimSpace(r.FormValue("time")),
		},
	}
	if job.Schedule.Type == schedule.TypeWeekly {
		job.Schedule.Weekdays = r.Form["weekdays"]
	}
	return job, r.FormValue("script_content"), nil
}

func parsePlainEnv(raw string) map[string]string {
	env := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			env[strings.TrimSpace(line)] = ""
			continue
		}
		env[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	return env
}

func splitCSV(raw string) []string {
	var out []string
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func findJob(cfg config.Config, id string) (config.Job, bool) {
	for _, job := range cfg.Jobs {
		if job.ID == id {
			return job, true
		}
	}
	return config.Job{}, false
}

func weekdays(selected []string) []weekdayOption {
	selectedMap := map[string]bool{}
	for _, day := range selected {
		selectedMap[day] = true
	}
	values := []weekdayOption{
		{Value: "sunday", Label: "Sun"},
		{Value: "monday", Label: "Mon"},
		{Value: "tuesday", Label: "Tue"},
		{Value: "wednesday", Label: "Wed"},
		{Value: "thursday", Label: "Thu"},
		{Value: "friday", Label: "Fri"},
		{Value: "saturday", Label: "Sat"},
	}
	for i := range values {
		values[i].Checked = selectedMap[values[i].Value]
	}
	return values
}

func envPlainText(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values[key])
	}
	return strings.Join(lines, "\n")
}

func defaultScriptText() string {
	return `#!/usr/bin/env bash
set -euo pipefail

echo "job=${JOB_ID}"
echo "reason=${JOB_RUN_REASON}"
`
}

var templates = template.Must(template.New("webui").Funcs(template.FuncMap{
	"join":         strings.Join,
	"envPlainText": envPlainText,
}).Parse(layoutTemplate + dashboardTemplate + jobFormTemplate + runLogTemplate))

const layoutTemplate = `
{{define "layout_start"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - Cron Jobs</title>
  <style>
    :root { color-scheme: light; --border:#d9dee7; --bg:#f6f7f9; --ink:#1f2937; --muted:#667085; --accent:#2f6fed; --danger:#b42318; }
    * { box-sizing: border-box; }
    body { margin:0; font:14px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; color:var(--ink); background:var(--bg); }
    header { height:52px; display:flex; align-items:center; justify-content:space-between; padding:0 24px; border-bottom:1px solid var(--border); background:white; }
    main { max-width:1180px; margin:0 auto; padding:24px; }
    h1 { font-size:20px; margin:0; }
    h2 { font-size:15px; margin:0 0 12px; }
    a { color:var(--accent); text-decoration:none; }
    .grid { display:grid; grid-template-columns: minmax(0, 1fr); gap:18px; }
    .panel { background:white; border:1px solid var(--border); border-radius:8px; padding:16px; }
    .toolbar { display:flex; gap:8px; align-items:center; justify-content:space-between; margin-bottom:12px; }
    table { width:100%; border-collapse:collapse; }
    th, td { text-align:left; padding:9px 8px; border-bottom:1px solid var(--border); vertical-align:middle; }
    th { font-size:12px; color:var(--muted); font-weight:600; }
    code, pre, textarea, input, select { font:13px ui-monospace,SFMono-Regular,Menlo,Consolas,monospace; }
    input, select, textarea { width:100%; border:1px solid var(--border); border-radius:6px; padding:8px; background:white; color:var(--ink); }
    textarea { min-height:110px; resize:vertical; }
    .script-editor { min-height:300px; }
    label { display:block; font-size:12px; color:var(--muted); margin-bottom:5px; }
    .fields { display:grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap:14px; }
    .wide { grid-column:1 / -1; }
    .checks { display:flex; flex-wrap:wrap; gap:10px; }
    .checks label { display:flex; align-items:center; gap:6px; margin:0; color:var(--ink); }
    .checks input { width:auto; }
    .btns { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
    button, .button { border:0; border-radius:6px; background:var(--accent); color:white; padding:8px 12px; cursor:pointer; display:inline-block; }
    .secondary { background:#eef1f5; color:var(--ink); }
    .danger { background:var(--danger); }
    .muted { color:var(--muted); }
    .next-run-countdown { display:block; margin-top:2px; }
    .next-run-countdown.due { color:var(--accent); }
    .error { padding:10px 12px; border:1px solid #fda29b; background:#fff1f0; color:#912018; border-radius:6px; margin-bottom:14px; }
    form.inline { display:inline; }
    pre { white-space:pre-wrap; background:#111827; color:#e5e7eb; padding:14px; border-radius:8px; overflow:auto; }
    @media (max-width: 760px) { main { padding:14px; } header { padding:0 14px; } .fields { grid-template-columns:1fr; } th:nth-child(3), td:nth-child(3) { display:none; } }
  </style>
</head>
<body>
<header>
  <h1>Cron Jobs</h1>
  <nav><a href="/">Dashboard</a> · <a href="/jobs/new">New Job</a></nav>
</header>
<main>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
{{end}}

{{define "layout_end"}}
</main>
<script>
(function () {
  var reloadTimer = null;
  var reloadDelayMs = 20000;

  function formatRemaining(ms) {
    if (!Number.isFinite(ms)) {
      return "";
    }
    if (ms <= 0) {
      return "checking...";
    }

    var seconds = Math.ceil(ms / 1000);
    if (seconds < 60) {
      return seconds + "s";
    }

    var minutes = Math.ceil(seconds / 60);
    if (minutes < 60) {
      return "~" + minutes + "m";
    }

    var hours = Math.floor(minutes / 60);
    var remainingMinutes = minutes % 60;
    if (remainingMinutes === 0) {
      return "~" + hours + "h";
    }
    return "~" + hours + "h " + remainingMinutes + "m";
  }

  function scheduleReload() {
    if (reloadTimer) {
      return;
    }
    reloadTimer = window.setTimeout(function () {
      window.location.reload();
    }, reloadDelayMs);
  }

  function updateNextRunCountdowns() {
    var now = Date.now();
    var hasDueJob = false;
    document.querySelectorAll("[data-next-run]").forEach(function (node) {
      var nextRun = Date.parse(node.dataset.nextRun);
      if (Number.isNaN(nextRun)) {
        node.textContent = "";
        return;
      }
      var remaining = nextRun - now;
      node.textContent = formatRemaining(remaining);
      if (remaining <= 0) {
        node.classList.add("due");
        hasDueJob = true;
      } else {
        node.classList.remove("due");
      }
    });
    if (hasDueJob) {
      scheduleReload();
    }
  }

  updateNextRunCountdowns();
  window.setInterval(updateNextRunCountdowns, 1000);
})();
</script>
</body>
</html>
{{end}}
`

const dashboardTemplate = `
{{define "dashboard"}}
{{template "layout_start" .}}
<div class="grid">
  <section class="panel">
    <div class="toolbar">
      <h2>Jobs</h2>
      <a class="button" href="/jobs/new">New Job</a>
    </div>
    <table>
      <thead><tr><th>Name</th><th>Schedule</th><th>Next Run</th><th>Status</th><th>Actions</th></tr></thead>
      <tbody>
      {{range .Jobs}}
        <tr>
          <td><a href="/jobs/{{.ID}}/edit">{{.Name}}</a></td>
          <td>{{.ScheduleType}}</td>
          <td>
            {{if .NextRun.IsZero}}
              -
            {{else}}
              <span>{{.NextRun.Format "2006-01-02 15:04:05 MST"}}</span>
              <span class="muted next-run-countdown" data-next-run="{{.NextRun.Format "2006-01-02T15:04:05Z07:00"}}" aria-live="polite"></span>
            {{end}}
          </td>
          <td>{{if .Running}}Running{{else if .Enabled}}Enabled{{else}}Disabled{{end}}</td>
          <td class="btns">
            <form class="inline" method="post" action="/jobs/{{.ID}}/run"><button type="submit">Run</button></form>
            <form class="inline" method="post" action="/jobs/{{.ID}}/toggle"><button class="secondary" type="submit">{{if .Enabled}}Disable{{else}}Enable{{end}}</button></form>
          </td>
        </tr>
      {{else}}
        <tr><td colspan="5" class="muted">No jobs configured.</td></tr>
      {{end}}
      </tbody>
    </table>
  </section>

  <section class="panel">
    <h2>Recent Runs</h2>
    <table>
      <thead><tr><th>Run</th><th>Job</th><th>Status</th><th>Finished</th></tr></thead>
      <tbody>
      {{range .Runs}}
        <tr>
          <td><a href="/runs/{{.RunID}}">{{.RunID}}</a></td>
          <td><a href="/jobs/{{.JobID}}/edit">{{.JobName}}</a></td>
          <td>{{.Status}}</td>
          <td>{{.FinishedAt.Format "2006-01-02 15:04:05 MST"}}</td>
        </tr>
      {{else}}
        <tr><td colspan="4" class="muted">No runs yet.</td></tr>
      {{end}}
      </tbody>
    </table>
  </section>

</div>
{{template "layout_end" .}}
{{end}}
`

const jobFormTemplate = `
{{define "job_form"}}
{{template "layout_start" .}}
<section class="panel">
  <div class="toolbar">
    <h2>{{.Title}}</h2>
    <a class="button secondary" href="/">Back</a>
  </div>
  <form method="post" action="{{.FormAction}}">
    <div class="fields">
      <div>
        <label>Name</label>
        <input name="name" value="{{.FormJob.Name}}">
      </div>
      <div>
        <label>Schedule Type</label>
        <select name="schedule_type">
          <option value="daily" {{if eq .FormJob.Schedule.Type "daily"}}selected{{end}}>Daily</option>
          <option value="weekly" {{if eq .FormJob.Schedule.Type "weekly"}}selected{{end}}>Weekly</option>
        </select>
      </div>
      <div>
        <label>Time</label>
        <input name="time" value="{{.FormJob.Schedule.Time}}" placeholder="18:10">
      </div>
      <div class="wide">
        <label>Weekdays</label>
        <div class="checks">
          {{range .WeekdayList}}
            <label><input type="checkbox" name="weekdays" value="{{.Value}}" {{if .Checked}}checked{{end}}> {{.Label}}</label>
          {{end}}
        </div>
      </div>
      <div>
        <label>Timeout Seconds</label>
        <input name="timeout_seconds" value="{{.FormJob.Runtime.TimeoutSeconds}}">
      </div>
      <div class="wide">
        <label>Script Content</label>
        <textarea class="script-editor" name="script_content" spellcheck="false">{{.ScriptText}}</textarea>
      </div>
      <div class="wide">
        <label>Plain Env</label>
        <textarea name="env_plain" placeholder="OWNER=itda-skills&#10;REPO=rs-golden-queens">{{envPlainText .FormJob.Env.Plain}}</textarea>
      </div>
      <div class="wide">
        <label>Inherited Env Names</label>
        <input name="env_inherit" value="{{join .FormJob.Env.Inherit ","}}" placeholder="GITHUB_PAT">
      </div>
      <div class="wide checks">
        <label><input type="checkbox" name="enabled" {{if .FormJob.Enabled}}checked{{end}}> Enabled</label>
      </div>
    </div>
    <p class="btns">
      <button class="secondary" type="submit" name="action" value="test">Run Test</button>
      <button type="submit" name="action" value="save">Save Job</button>
    </p>
    {{if .TestStatus}}
      <h2>Test Output - {{.TestStatus}}</h2>
      <pre>{{.TestOutput}}</pre>
    {{end}}
  </form>
</section>
{{template "layout_end" .}}
{{end}}
`

const runLogTemplate = `
{{define "run_log"}}
{{template "layout_start" .}}
<section class="panel">
  <div class="toolbar">
    <h2>Run Log</h2>
    <a class="button secondary" href="/">Back</a>
  </div>
  <p>
    <span class="muted">Run</span> {{.RunID}}<br>
    <span class="muted">Job</span> <a href="/jobs/{{.Run.JobID}}/edit">{{.Run.JobName}}</a>
  </p>
  <pre>{{.RunLog}}</pre>
</section>
{{template "layout_end" .}}
{{end}}
`
