package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ShutdownTimeout = 10 * time.Second

type Settings struct {
	Addr       string
	DataDir    string
	ConfigPath string
	LogDir     string
	ScriptDir  string
	Timezone   string
}

func LoadSettingsFromEnv() Settings {
	dataDir := envDefault("APP_DATA_DIR", "/data")

	return Settings{
		Addr:       envDefault("APP_ADDR", ":8080"),
		DataDir:    dataDir,
		ConfigPath: envDefault("APP_CONFIG_PATH", filepath.Join(dataDir, "config.json")),
		LogDir:     envDefault("APP_LOG_DIR", filepath.Join(dataDir, "logs")),
		ScriptDir:  envDefault("APP_SCRIPT_DIR", filepath.Join(dataDir, "scripts", "jobs")),
		Timezone:   envDefault("APP_TIMEZONE", "Asia/Seoul"),
	}
}

func (s Settings) Validate() error {
	if strings.TrimSpace(s.Addr) == "" {
		return errors.New("APP_ADDR is required")
	}
	if strings.TrimSpace(s.DataDir) == "" {
		return errors.New("APP_DATA_DIR is required")
	}
	if strings.TrimSpace(s.ConfigPath) == "" {
		return errors.New("APP_CONFIG_PATH is required")
	}
	if strings.TrimSpace(s.LogDir) == "" {
		return errors.New("APP_LOG_DIR is required")
	}
	if strings.TrimSpace(s.ScriptDir) == "" {
		return errors.New("APP_SCRIPT_DIR is required")
	}
	if strings.TrimSpace(s.Timezone) == "" {
		return errors.New("APP_TIMEZONE is required")
	}
	return nil
}

func envDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}
