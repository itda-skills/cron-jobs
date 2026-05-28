package jobruntime

import (
	"path/filepath"
	"testing"
)

func TestResolveRuntimeScript(t *testing.T) {
	dataDir := t.TempDir()
	paths := Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
	}

	got, err := Resolve(
		Config{
			Language:       LanguageBash,
			Script:         "scripts/jobs/report.sh",
			TimeoutSeconds: 60,
		},
		paths,
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Script != filepath.Join(paths.ScriptDir, "report.sh") {
		t.Fatalf("Script = %q, want under script dir", got.Script)
	}
}

func TestResolveRejectsScriptOutsideScriptDir(t *testing.T) {
	dataDir := t.TempDir()
	paths := Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
	}

	_, err := Resolve(Config{
		Language:       LanguageBash,
		Script:         "../secret.sh",
		TimeoutSeconds: 60,
	}, paths)
	if err == nil {
		t.Fatal("Resolve() error = nil for script outside script dir")
	}
}

func TestResolveRejectsUnsupportedLanguage(t *testing.T) {
	dataDir := t.TempDir()
	_, err := Resolve(Config{
		Language:       "python",
		Script:         "scripts/jobs/report.py",
		TimeoutSeconds: 60,
	}, Paths{DataDir: dataDir, ScriptDir: filepath.Join(dataDir, "scripts", "jobs")})
	if err == nil {
		t.Fatal("Resolve() error = nil for unsupported language")
	}
}
