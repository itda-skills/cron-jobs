package jobenv

import "testing"

func TestMergeAppliesGlobalAndJobEnv(t *testing.T) {
	got, err := Merge(
		Config{
			Plain: map[string]string{"BRANCH": "main", "SHARED": "global"},
			Inherit: []string{
				"GITHUB_PAT",
			},
		},
		Config{
			Plain:   map[string]string{"SHARED": "job", "OWNER": "itda-skills"},
			Inherit: []string{"JOB_TOKEN"},
		},
		func(name string) (string, bool) {
			values := map[string]string{
				"GITHUB_PAT": "global-secret",
				"JOB_TOKEN":  "job-secret",
			}
			value, ok := values[name]
			return value, ok
		},
	)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	want := map[string]string{
		"BRANCH":     "main",
		"SHARED":     "job",
		"OWNER":      "itda-skills",
		"GITHUB_PAT": "global-secret",
		"JOB_TOKEN":  "job-secret",
	}
	for name, value := range want {
		if got[name] != value {
			t.Fatalf("env[%s] = %q, want %q", name, got[name], value)
		}
	}
}

func TestMergeReportsMissingInheritedEnv(t *testing.T) {
	_, err := Merge(Config{Inherit: []string{"GITHUB_PAT"}}, Config{}, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("Merge() error = nil, want missing inherited env error")
	}
}

func TestValidateRejectsInvalidNames(t *testing.T) {
	if err := Validate("job", Config{Plain: map[string]string{"BAD-NAME": "x"}}); err == nil {
		t.Fatal("Validate() error = nil for invalid plain env name")
	}
	if err := Validate("job", Config{Inherit: []string{"TOKEN=value"}}); err == nil {
		t.Fatal("Validate() error = nil for invalid inherited env name")
	}
}
