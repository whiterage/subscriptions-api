package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	APIKey      string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		APIKey:      os.Getenv("API_KEY"),
	}

	if cfg.DatabaseURL == "" {
		fileCfg, err := loadYAML("config.yaml")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
		if cfg.HTTPAddr == ":8080" && fileCfg.HTTPAddr != "" {
			cfg.HTTPAddr = fileCfg.HTTPAddr
		}
		cfg.DatabaseURL = fileCfg.DatabaseURL
		cfg.APIKey = fileCfg.APIKey
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.APIKey == "" {
		return Config{}, errors.New("API_KEY is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func loadYAML(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	section := ""
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return Config{}, fmt.Errorf("invalid config line: %q", rawLine)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		switch section + "." + key {
		case "http.addr":
			cfg.HTTPAddr = value
		case "database.url":
			cfg.DatabaseURL = value
		case "auth.api_key":
			cfg.APIKey = value
		}
	}

	return cfg, nil
}
