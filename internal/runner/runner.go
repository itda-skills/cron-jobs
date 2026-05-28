package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/logstore"
)

const (
	RunReasonScheduled = "scheduled"
	RunReasonManual    = "manual"
	RunReasonTest      = "test"
)

type Runner struct {
	Store       logstore.Store
	ConfigPath  string
	WorkDirRoot string
}

type Job struct {
	ID           string
	Name         string
	ScheduleType string
	ScheduledAt  time.Time
	RunReason    string
	Runtime      jobruntime.Resolved
	Env          map[string]string
}

func (r Runner) Run(ctx context.Context, job Job) (logstore.Entry, error) {
	runID, relLogPath, logFile, err := r.Store.CreateRunLog(job.ID, job.ScheduledAt)
	if err != nil {
		return logstore.Entry{}, err
	}
	defer logFile.Close()

	startedAt := time.Now()
	entry := logstore.Entry{
		RunID:       runID,
		JobID:       job.ID,
		JobName:     job.Name,
		ScheduledAt: job.ScheduledAt,
		StartedAt:   startedAt,
		ExitCode:    -1,
		LogPath:     relLogPath,
	}

	workDir, cleanup, err := r.createWorkDir(runID)
	if err != nil {
		entry = r.finish(entry, startedAt, -1, logstore.StatusFailed, err)
		_ = r.Store.Append(entry)
		return entry, err
	}
	defer cleanup()

	wrapperPath, err := writeBashWrapper(workDir, job.Runtime)
	if err != nil {
		entry = r.finish(entry, startedAt, -1, logstore.StatusFailed, err)
		_ = r.Store.Append(entry)
		return entry, err
	}

	timeout := job.Runtime.Timeout
	if timeout <= 0 {
		timeout = time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "/usr/bin/env", "bash", wrapperPath)
	cmd.Dir = workDir
	cmd.Env = r.buildEnv(job, workDir)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Run()
	status := logstore.StatusSuccess
	exitCode := 0
	if runCtx.Err() == context.DeadlineExceeded {
		status = logstore.StatusTimedOut
		exitCode = -1
		err = fmt.Errorf("job timed out after %s", timeout)
	} else if err != nil {
		status = logstore.StatusFailed
		exitCode = exitCodeFromError(err)
	}

	entry = r.finish(entry, startedAt, exitCode, status, err)
	if appendErr := r.Store.Append(entry); appendErr != nil && err == nil {
		err = appendErr
	}
	return entry, err
}

func (r Runner) finish(entry logstore.Entry, startedAt time.Time, exitCode int, status string, err error) logstore.Entry {
	finishedAt := time.Now()
	entry.StartedAt = startedAt
	entry.FinishedAt = finishedAt
	entry.DurationMillis = finishedAt.Sub(startedAt).Milliseconds()
	entry.ExitCode = exitCode
	entry.Status = status
	if err != nil {
		entry.Error = err.Error()
	}
	return entry
}

func (r Runner) createWorkDir(runID string) (string, func(), error) {
	root := r.WorkDirRoot
	if root == "" {
		root = os.TempDir()
	}
	dir, err := os.MkdirTemp(root, "cron-jobs-"+runID+"-")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

func (r Runner) buildEnv(job Job, workDir string) []string {
	runReason := job.RunReason
	if runReason == "" {
		runReason = RunReasonScheduled
	}
	testRun := "false"
	if runReason == RunReasonTest {
		testRun = "true"
	}
	env := map[string]string{
		"PATH":              "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME":              workDir,
		"TMPDIR":            workDir,
		"JOB_ID":            job.ID,
		"JOB_NAME":          job.Name,
		"JOB_SCHEDULE_TYPE": job.ScheduleType,
		"JOB_RUN_AT":        job.ScheduledAt.Format(time.RFC3339),
		"JOB_CONFIG_PATH":   r.ConfigPath,
		"JOB_LOG_DIR":       r.Store.Dir,
		"JOB_SCRIPT_PATH":   job.Runtime.Script,
		"JOB_RUN_REASON":    runReason,
		"JOB_TEST_RUN":      testRun,
	}
	for name, value := range job.Env {
		env[name] = value
	}

	out := make([]string, 0, len(env))
	for name, value := range env {
		out = append(out, name+"="+value)
	}
	return out
}

func writeBashWrapper(workDir string, runtime jobruntime.Resolved) (string, error) {
	if runtime.Language != jobruntime.LanguageBash {
		return "", fmt.Errorf("unsupported runtime language %q", runtime.Language)
	}
	path := filepath.Join(workDir, "job-wrapper.sh")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString("#!/usr/bin/env bash\nset -euo pipefail\n"); err != nil {
		return "", err
	}
	if _, err := file.WriteString("source \"$JOB_SCRIPT_PATH\"\n"); err != nil {
		return "", err
	}
	return path, nil
}

func exitCodeFromError(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		status, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok {
			return status.ExitStatus()
		}
	}
	return -1
}
