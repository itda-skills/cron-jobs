package scriptstore

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var safeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type Store struct {
	DataDir   string
	ScriptDir string
}

func (s Store) PathForJob(jobID string) (string, string, error) {
	if !safeIDPattern.MatchString(jobID) {
		return "", "", fmt.Errorf("job id %q contains unsupported characters", jobID)
	}
	abs := filepath.Join(s.ScriptDir, jobID+".sh")
	rel, err := filepath.Rel(s.DataDir, abs)
	if err != nil {
		return "", "", err
	}
	return filepath.ToSlash(rel), abs, nil
}

func (s Store) WriteJob(jobID string, content string) (string, error) {
	rel, abs, err := s.PathForJob(jobID)
	if err != nil {
		return "", err
	}
	if err := writeFile(abs, content); err != nil {
		return "", err
	}
	return rel, nil
}

func (s Store) ReadConfigured(configured string) (string, error) {
	abs, err := s.resolveConfigured(configured)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s Store) WriteTest(jobID string, content string, now time.Time) (string, func(), error) {
	if !safeIDPattern.MatchString(jobID) {
		return "", func() {}, fmt.Errorf("job id %q contains unsupported characters", jobID)
	}
	name := fmt.Sprintf("%s-%s.sh", jobID, now.UTC().Format("20060102T150405.000000000Z"))
	abs := filepath.Join(s.ScriptDir, ".tests", name)
	if err := writeFile(abs, content); err != nil {
		return "", func() {}, err
	}
	rel, err := filepath.Rel(s.DataDir, abs)
	if err != nil {
		return "", func() {}, err
	}
	return filepath.ToSlash(rel), func() { _ = os.Remove(abs) }, nil
}

func (s Store) resolveConfigured(configured string) (string, error) {
	if configured == "" {
		return "", fmt.Errorf("script path is required")
	}
	var candidate string
	if filepath.IsAbs(configured) {
		candidate = configured
	} else {
		candidate = filepath.Join(s.DataDir, configured)
	}
	baseAbs, err := filepath.Abs(filepath.Clean(s.ScriptDir))
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
	if rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("script path %q must be inside %q", configured, s.ScriptDir)
	}
	return candidateAbs, nil
}

func writeFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content = normalizeLineEndings(content)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".script-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o700); err != nil {
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

func normalizeLineEndings(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}
