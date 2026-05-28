package jobruntime

import (
	"fmt"
	"path/filepath"
	"time"
)

const LanguageBash = "bash"

type Config struct {
	Language       string   `json:"language"`
	Script         string   `json:"script"`
	Recipes        []string `json:"recipes,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type Recipe struct {
	ID          string `json:"id"`
	Language    string `json:"language"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Version     string `json:"version,omitempty"`
}

type Paths struct {
	DataDir   string
	ScriptDir string
	RecipeDir string
}

type Resolved struct {
	Language string
	Script   string
	Recipes  []ResolvedRecipe
	Timeout  time.Duration
}

type ResolvedRecipe struct {
	ID   string
	Path string
}

func ValidateLanguage(language string) error {
	if language != LanguageBash {
		return fmt.Errorf("unsupported runtime language %q", language)
	}
	return nil
}

func Resolve(runtime Config, catalog []Recipe, paths Paths) (Resolved, error) {
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

	recipesByID := make(map[string]Recipe, len(catalog))
	for _, recipe := range catalog {
		if _, exists := recipesByID[recipe.ID]; exists {
			return Resolved{}, fmt.Errorf("duplicate recipe id %q", recipe.ID)
		}
		if err := ValidateLanguage(recipe.Language); err != nil {
			return Resolved{}, fmt.Errorf("recipe %q: %w", recipe.ID, err)
		}
		if _, err := ResolveUnder(paths.DataDir, paths.RecipeDir, recipe.Path); err != nil {
			return Resolved{}, fmt.Errorf("recipe %q has invalid path: %w", recipe.ID, err)
		}
		recipesByID[recipe.ID] = recipe
	}

	resolvedRecipes := make([]ResolvedRecipe, 0, len(runtime.Recipes))
	seenSelected := map[string]struct{}{}
	for _, id := range runtime.Recipes {
		if _, ok := seenSelected[id]; ok {
			return Resolved{}, fmt.Errorf("duplicate selected recipe %q", id)
		}
		seenSelected[id] = struct{}{}

		recipe, ok := recipesByID[id]
		if !ok {
			return Resolved{}, fmt.Errorf("selected recipe %q does not exist", id)
		}
		if recipe.Language != runtime.Language {
			return Resolved{}, fmt.Errorf("selected recipe %q language %q does not match runtime %q", id, recipe.Language, runtime.Language)
		}
		path, err := ResolveUnder(paths.DataDir, paths.RecipeDir, recipe.Path)
		if err != nil {
			return Resolved{}, fmt.Errorf("recipe %q has invalid path: %w", id, err)
		}
		resolvedRecipes = append(resolvedRecipes, ResolvedRecipe{ID: id, Path: path})
	}

	return Resolved{
		Language: runtime.Language,
		Script:   scriptPath,
		Recipes:  resolvedRecipes,
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
