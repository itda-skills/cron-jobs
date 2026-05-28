package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/logstore"
	"github.com/itda-skills/cron-jobs/internal/runner"
	"github.com/itda-skills/cron-jobs/internal/schedule"
	"github.com/itda-skills/cron-jobs/internal/scheduler"
)

type Service struct {
	settings Settings
	paths    config.Paths
	store    logstore.Store
	runner   runner.Runner
	lookup   func(string) (string, bool)

	mu      sync.Mutex
	cfg     config.Config
	jobs    map[string]scheduler.PlannedJob
	running map[string]bool
}

type JobView struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Enabled      bool      `json:"enabled"`
	ScheduleType string    `json:"schedule_type"`
	NextRun      time.Time `json:"next_run"`
	Running      bool      `json:"running"`
}

func NewService(settings Settings) *Service {
	paths := config.Paths{
		DataDir:   settings.DataDir,
		ScriptDir: settings.ScriptDir,
	}
	store := logstore.Store{Dir: settings.LogDir}
	return &Service{
		settings: settings,
		paths:    paths,
		store:    store,
		runner: runner.Runner{
			Store:      store,
			ConfigPath: settings.ConfigPath,
		},
		lookup:  os.LookupEnv,
		jobs:    map[string]scheduler.PlannedJob{},
		running: map[string]bool{},
	}
}

func (s *Service) Load() error {
	cfg, err := config.Load(s.settings.ConfigPath)
	if os.IsNotExist(err) {
		cfg = config.Config{
			Version:  1,
			Timezone: s.settings.Timezone,
			Jobs:     []config.Job{},
		}
		if err := config.Save(s.settings.ConfigPath, cfg); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return s.applyConfig(cfg, time.Now())
}

func (s *Service) Config() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *Service) SaveConfig(cfg config.Config) error {
	if err := cfg.Validate(s.paths); err != nil {
		return err
	}
	if err := config.Save(s.settings.ConfigPath, cfg); err != nil {
		return err
	}
	return s.applyConfig(cfg, time.Now())
}

func (s *Service) ListJobs() []JobView {
	s.mu.Lock()
	defer s.mu.Unlock()

	views := make([]JobView, 0, len(s.cfg.Jobs))
	for _, job := range s.cfg.Jobs {
		view := JobView{
			ID:           job.ID,
			Name:         job.Name,
			Enabled:      job.Enabled,
			ScheduleType: job.Schedule.Type,
			Running:      s.running[job.ID],
		}
		if planned, ok := s.jobs[job.ID]; ok {
			view.NextRun = planned.NextRun
		}
		views = append(views, view)
	}
	return views
}

func (s *Service) RecentRuns(limit int) ([]logstore.Entry, error) {
	return s.store.Recent(limit)
}

func (s *Service) FindRun(runID string) (logstore.Entry, error) {
	entries, err := s.store.Recent(0)
	if err != nil {
		return logstore.Entry{}, err
	}
	for _, entry := range entries {
		if entry.RunID == runID {
			return entry, nil
		}
	}
	return logstore.Entry{}, fmt.Errorf("run %q not found", runID)
}

func (s *Service) ReadRunLog(runID string) ([]byte, error) {
	entry, err := s.FindRun(runID)
	if err != nil {
		return nil, err
	}
	return s.store.ReadLog(entry)
}

func (s *Service) RunJobNow(ctx context.Context, id string) (logstore.Entry, error) {
	planned, err := s.reserveJob(id, time.Now())
	if err != nil {
		return logstore.Entry{}, err
	}
	defer s.releaseJob(id)

	return s.runner.Run(ctx, runner.Job{
		ID:           planned.ID,
		Name:         planned.Name,
		ScheduleType: planned.ScheduleType,
		ScheduledAt:  time.Now(),
		RunReason:    runner.RunReasonManual,
		Runtime:      planned.Runtime,
		Env:          planned.Env,
	})
}

func (s *Service) RunDue(ctx context.Context, now time.Time) {
	due := s.reserveDue(now)
	for _, planned := range due {
		go func(job scheduler.PlannedJob) {
			defer s.releaseJob(job.ID)
			_, _ = s.runner.Run(ctx, runner.Job{
				ID:           job.ID,
				Name:         job.Name,
				ScheduleType: job.ScheduleType,
				ScheduledAt:  job.NextRun,
				RunReason:    runner.RunReasonScheduled,
				Runtime:      job.Runtime,
				Env:          job.Env,
			})
		}(planned)
	}
}

func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.RunDue(ctx, now)
		}
	}
}

func (s *Service) applyConfig(cfg config.Config, now time.Time) error {
	planned, err := scheduler.BuildPlan(cfg, s.paths, now, s.lookup)
	if err != nil {
		return err
	}
	jobs := make(map[string]scheduler.PlannedJob, len(planned))
	for _, job := range planned {
		jobs[job.ID] = job
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	s.jobs = jobs
	if s.running == nil {
		s.running = map[string]bool{}
	}
	return nil
}

func (s *Service) reserveJob(id string, scheduledAt time.Time) (scheduler.PlannedJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	planned, ok := s.jobs[id]
	if !ok {
		return scheduler.PlannedJob{}, fmt.Errorf("job %q not found or disabled", id)
	}
	if s.running[id] {
		return scheduler.PlannedJob{}, fmt.Errorf("job %q is already running", id)
	}
	planned.NextRun = scheduledAt
	s.running[id] = true
	return planned, nil
}

func (s *Service) reserveDue(now time.Time) []scheduler.PlannedJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	var due []scheduler.PlannedJob
	for id, planned := range s.jobs {
		if planned.NextRun.IsZero() || planned.NextRun.After(now) {
			continue
		}
		if s.running[id] {
			_ = s.store.Append(logstore.Entry{
				RunID:       logstore.NewRunID(id, planned.NextRun),
				JobID:       id,
				JobName:     planned.Name,
				ScheduledAt: planned.NextRun,
				StartedAt:   now,
				FinishedAt:  now,
				ExitCode:    -1,
				Status:      logstore.StatusSkipped,
				Error:       "previous run is still active",
			})
			next, err := schedule.NextRun(planned.Schedule, planned.NextRun, planned.NextRun.Location())
			if err == nil {
				planned.NextRun = next
				s.jobs[id] = planned
			}
			continue
		}
		s.running[id] = true
		due = append(due, planned)
		next, err := schedule.NextRun(planned.Schedule, planned.NextRun, planned.NextRun.Location())
		if err == nil {
			planned.NextRun = next
			s.jobs[id] = planned
		}
	}
	return due
}

func (s *Service) releaseJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running == nil {
		return
	}
	delete(s.running, id)
}

func IsNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
