package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/itda-skills/cron-jobs/internal/jobenv"
	"github.com/itda-skills/cron-jobs/internal/jobruntime"
	"github.com/itda-skills/cron-jobs/internal/schedule"
)

var jobIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type Config struct {
	Version  int        `json:"version"`
	Timezone string     `json:"timezone"`
	Env      EnvSection `json:"env,omitempty"`
	Jobs     []Job      `json:"jobs"`
}

type EnvSection struct {
	Global jobenv.Config `json:"global,omitempty"`
}

type Job struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Enabled  bool              `json:"enabled"`
	Env      jobenv.Config     `json:"env,omitempty"`
	Runtime  jobruntime.Config `json:"runtime"`
	Schedule schedule.Spec     `json:"schedule"`
}

type Paths struct {
	DataDir   string
	ScriptDir string
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (c Config) Validate(paths Paths) error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if c.Timezone == "" {
		return errors.New("timezone is required")
	}
	if err := jobenv.Validate("global", c.Env.Global); err != nil {
		return err
	}

	runtimePaths := jobruntime.Paths{
		DataDir:   paths.DataDir,
		ScriptDir: paths.ScriptDir,
	}

	jobIDs := map[string]struct{}{}
	for _, job := range c.Jobs {
		if job.ID == "" {
			return errors.New("job id is required")
		}
		if !jobIDPattern.MatchString(job.ID) {
			return fmt.Errorf("job id %q contains unsupported characters", job.ID)
		}
		if _, ok := jobIDs[job.ID]; ok {
			return fmt.Errorf("duplicate job id %q", job.ID)
		}
		jobIDs[job.ID] = struct{}{}
		if job.Name == "" {
			return fmt.Errorf("job %q name is required", job.ID)
		}
		if err := jobenv.Validate("job "+job.ID, job.Env); err != nil {
			return err
		}
		if _, err := jobruntime.Resolve(job.Runtime, runtimePaths); err != nil {
			return fmt.Errorf("job %q runtime: %w", job.ID, err)
		}
		if err := schedule.Validate(job.Schedule); err != nil {
			return fmt.Errorf("job %q schedule: %w", job.ID, err)
		}
	}
	return nil
}
