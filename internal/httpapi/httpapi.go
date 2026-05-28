package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/itda-skills/cron-jobs/internal/app"
	"github.com/itda-skills/cron-jobs/internal/config"
)

type Server struct {
	Service *app.Service
}

func (s Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.getConfig)
	mux.HandleFunc("PUT /api/config", s.putConfig)
	mux.HandleFunc("GET /api/jobs", s.listJobs)
	mux.HandleFunc("POST /api/jobs/test", s.testJob)
	mux.HandleFunc("POST /api/jobs/{id}/run", s.runJob)
	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("GET /api/runs/{id}/log", s.getRunLog)
	return mux
}

type testJobRequest struct {
	Job           config.Job `json:"job"`
	ScriptContent string     `json:"script_content"`
}

type testJobResponse struct {
	Entry  any    `json:"entry"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func (s Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Service.Config())
}

func (s Server) putConfig(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var cfg config.Config
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Service.SaveConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.Service.Config())
}

func (s Server) listJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Service.ListJobs())
}

func (s Server) runJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	entry, err := s.Service.RunJobNow(ctx, id)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s Server) testJob(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req testJobRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	entry, output, err := s.Service.TestJob(ctx, req.Job, req.ScriptContent)
	resp := testJobResponse{Entry: entry, Output: output}
	if err != nil {
		if entry.RunID == "" {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		resp.Error = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) listRuns(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be a non-negative integer"))
			return
		}
		limit = parsed
	}
	entries, err := s.Service.RecentRuns(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s Server) getRunLog(w http.ResponseWriter, r *http.Request) {
	data, err := s.Service.ReadRunLog(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
