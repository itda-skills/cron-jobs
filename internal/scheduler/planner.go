package scheduler

import (
	"fmt"
	"time"

	"github.com/itda-skills/cron-jobs/internal/config"
	"github.com/itda-skills/cron-jobs/internal/jobenv"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

type PlannedJob struct {
	ID           string
	Name         string
	ScheduleType string
	NextRun      time.Time
	Runtime      jobruntime.Resolved
	Env          map[string]string
}

func BuildPlan(cfg config.Config, paths config.Paths, now time.Time, lookup func(string) (string, bool)) ([]PlannedJob, error) {
	if err := cfg.Validate(paths); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", cfg.Timezone, err)
	}
	runtimePaths := jobruntime.Paths{
		DataDir:   paths.DataDir,
		ScriptDir: paths.ScriptDir,
		RecipeDir: paths.RecipeDir,
	}

	planned := make([]PlannedJob, 0, len(cfg.Jobs))
	for _, job := range cfg.Jobs {
		if !job.Enabled {
			continue
		}
		env, err := jobenv.Merge(cfg.Env.Global, job.Env, lookup)
		if err != nil {
			return nil, fmt.Errorf("job %q env: %w", job.ID, err)
		}
		resolved, err := jobruntime.Resolve(job.Runtime, cfg.Recipes, runtimePaths)
		if err != nil {
			return nil, fmt.Errorf("job %q runtime: %w", job.ID, err)
		}
		nextRun, err := schedule.NextRun(job.Schedule, now, loc)
		if err != nil {
			return nil, fmt.Errorf("job %q schedule: %w", job.ID, err)
		}
		planned = append(planned, PlannedJob{
			ID:           job.ID,
			Name:         job.Name,
			ScheduleType: job.Schedule.Type,
			NextRun:      nextRun,
			Runtime:      resolved,
			Env:          env,
		})
	}
	return planned, nil
}
