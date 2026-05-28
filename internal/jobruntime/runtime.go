package jobruntime

import (
	"fmt"
	"path/filepath"
	"time"
)

const LanguageBash = "bash"

type Config struct {
	Language       string `json:"language"`
	Script         string `json:"script"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type Paths struct {
	DataDir   string
	ScriptDir string
}

type Resolved struct {
	Language string
	Script   string
	Timeout  time.Duration
}

func ValidateLanguage(language string) error {
	if language != LanguageBash {
		return fmt.Errorf("unsupported runtime language %q", language)
	}
	return nil
}

func Resolve(runtime Config, paths Paths) (Resolved, error) {
	if err := ValidateLanguage(runtime.Language); err != nil {
		return Resolved{}, err
	}
	if runtime.TimeoutSeconds <= 0 {
		return Resolved{}, fmt.Errorf("timeout_seconds must be greater than zero")
	}
	scriptPath, err := ResolveUnder(paths.DataDir, paths.ScriptDir, runtime.Script)
	if err != nil {
		return Resolved{}, fmt.Errorf("invalid script path: %w", err)
	}

	return Resolved{
		Language: runtime.Language,
		Script:   scriptPath,
		Timeout:  time.Duration(runtime.TimeoutSeconds) * time.Second,
	}, nil
}

func ResolveUnder(dataDir string, baseDir string, configured string) (string, error) {
	if configured == "" {
		return "", fmt.Errorf("path is required")
	}

	var candidate string
	if filepath.IsAbs(configured) {
		candidate = configured
	} else {
		candidate = filepath.Join(dataDir, configured)
	}

	baseAbs, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", err
	}
	candidateAbs, err := filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || rel == "" {
		return "", fmt.Errorf("path %q must be inside %q", configured, baseDir)
	}
	if len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("path %q must be inside %q", configured, baseDir)
	}

	return candidateAbs, nil
}
