package jobruntime

import (
	"path/filepath"
	"testing"
)

func TestResolveRuntimeAndRecipes(t *testing.T) {
	dataDir := t.TempDir()
	paths := Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
		RecipeDir: filepath.Join(dataDir, "recipes"),
	}

	got, err := Resolve(
		Config{
			Language:       LanguageBash,
			Script:         "scripts/jobs/report.sh",
			Recipes:        []string{"github-actions"},
			TimeoutSeconds: 60,
		},
		[]Recipe{{
			ID:       "github-actions",
			Language: LanguageBash,
			Path:     "recipes/bash/github-actions.sh",
		}},
		paths,
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Script != filepath.Join(paths.ScriptDir, "report.sh") {
		t.Fatalf("Script = %q, want under script dir", got.Script)
	}
	if len(got.Recipes) != 1 || got.Recipes[0].ID != "github-actions" {
		t.Fatalf("Recipes = %#v, want github-actions", got.Recipes)
	}
}

func TestResolveRejectsScriptOutsideScriptDir(t *testing.T) {
	dataDir := t.TempDir()
	paths := Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
		RecipeDir: filepath.Join(dataDir, "recipes"),
	}

	_, err := Resolve(Config{
		Language:       LanguageBash,
		Script:         "../secret.sh",
		TimeoutSeconds: 60,
	}, nil, paths)
	if err == nil {
		t.Fatal("Resolve() error = nil for script outside script dir")
	}
}

func TestResolveRejectsMissingRecipe(t *testing.T) {
	dataDir := t.TempDir()
	paths := Paths{
		DataDir:   dataDir,
		ScriptDir: filepath.Join(dataDir, "scripts", "jobs"),
		RecipeDir: filepath.Join(dataDir, "recipes"),
	}

	_, err := Resolve(Config{
		Language:       LanguageBash,
		Script:         "scripts/jobs/report.sh",
		Recipes:        []string{"missing"},
		TimeoutSeconds: 60,
	}, nil, paths)
	if err == nil {
		t.Fatal("Resolve() error = nil for missing recipe")
	}
}
