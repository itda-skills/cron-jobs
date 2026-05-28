package jobenv

import (
	"fmt"
	"regexp"
)

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Config struct {
	Plain   map[string]string `json:"plain,omitempty"`
	Inherit []string          `json:"inherit,omitempty"`
}

func ValidName(name string) bool {
	return envNamePattern.MatchString(name)
}

func Validate(scope string, cfg Config) error {
	for name := range cfg.Plain {
		if !ValidName(name) {
			return fmt.Errorf("%s plain env has invalid name %q", scope, name)
		}
	}
	for _, name := range cfg.Inherit {
		if !ValidName(name) {
			return fmt.Errorf("%s inherited env has invalid name %q", scope, name)
		}
	}
	return nil
}

func Merge(global Config, job Config, lookup func(string) (string, bool)) (map[string]string, error) {
	if err := Validate("global", global); err != nil {
		return nil, err
	}
	if err := Validate("job", job); err != nil {
		return nil, err
	}

	env := make(map[string]string, len(global.Plain)+len(job.Plain)+len(global.Inherit)+len(job.Inherit))
	for name, value := range global.Plain {
		env[name] = value
	}
	for name, value := range job.Plain {
		env[name] = value
	}

	seen := map[string]struct{}{}
	for _, name := range append(global.Inherit, job.Inherit...) {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		value, ok := lookup(name)
		if !ok {
			return nil, fmt.Errorf("inherited env %q is not set", name)
		}
		env[name] = value
	}

	return env, nil
}
