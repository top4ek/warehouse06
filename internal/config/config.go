package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `yaml:"env" env:"ENV" env-default:"development"` // development, production
	StorageDir string `yaml:"storage_dir" env:"STORAGE_DIR" env-default:"storage"`
	StorageURL string `yaml:"storage_url" env:"STORAGE_URL"`
	Server     `yaml:"server"`
	Database   `yaml:"database"`
	Sync       `yaml:"sync"`
}

type Server struct {
	Port int `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
}

type Database struct {
	Driver string `yaml:"driver" env-default:"sqlite3"` // sqlite3, postgres
	DSN    string `yaml:"dsn" env:"DATABASE_DSN" env-default:":memory:"`
}

type Sync struct {
	IntervalHours int `yaml:"interval_hours" env:"SYNC_INTERVAL_HOURS" env-default:"0"`
}

// SyncInterval returns the periodic sync interval. Zero means startup-only sync.
func (c *Config) SyncInterval() time.Duration {
	if c.IntervalHours <= 0 {
		return 0
	}
	return time.Duration(c.IntervalHours) * time.Hour
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	var cfg Config

	// cleanenv fails to parse typed fields (e.g. int) when the env var is present
	// but empty (SERVER_PORT=""). Treat empty values as unset so defaults apply,
	// restoring the environment afterwards so loading has no lasting side effects.
	restoreEnv := unsetIfEmpty(
		"ENV",
		"STORAGE_DIR",
		"STORAGE_URL",
		"SERVER_PORT",
		"DATABASE_DSN",
		"SYNC_INTERVAL_HOURS",
	)
	defer restoreEnv()

	if configPath == "" {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("cannot read env variables: %s", err)
		}
		return normalize(&cfg)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("config file %s does not exist, using env and default values", configPath)
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("cannot read env variables: %s", err)
		}
		return normalize(&cfg)
	} else if err != nil {
		log.Fatalf("cannot stat config file %s: %s", configPath, err)
	}

	// Precedence: defaults (env-default) → YAML → env (env wins).
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return normalize(&cfg)
}

func unsetIfEmpty(keys ...string) (restore func()) {
	unset := make([]string, 0, len(keys))
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok && v == "" {
			unset = append(unset, key)
			_ = os.Unsetenv(key)
		}
	}
	return func() {
		for _, key := range unset {
			_ = os.Setenv(key, "")
		}
	}
}

func normalize(cfg *Config) *Config {
	if cfg.StorageDir != "" {
		abs, err := filepath.Abs(cfg.StorageDir)
		if err != nil {
			log.Fatalf("cannot resolve storage_dir %q: %s", cfg.StorageDir, err)
		}
		cfg.StorageDir = abs
	}
	if err := validateStorageURL(cfg.StorageURL); err != nil {
		log.Fatalf("invalid storage_url: %s", err)
	}
	return cfg
}

// validateStorageURL rejects values that git would interpret as options or
// that enable command execution via remote helpers (ext::, fd::).
func validateStorageURL(url string) error {
	if url == "" {
		return nil
	}
	if strings.HasPrefix(url, "-") {
		return fmt.Errorf("must not start with %q: %q", "-", url)
	}
	lower := strings.ToLower(url)
	if strings.HasPrefix(lower, "ext::") || strings.HasPrefix(lower, "fd::") {
		return fmt.Errorf("transport helper URLs are not allowed: %q", url)
	}
	return nil
}
