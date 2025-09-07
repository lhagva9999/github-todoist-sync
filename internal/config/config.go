package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHub  GitHubConfig
	Todoist TodoistConfig
	App     AppConfig
}

type GitHubConfig struct {
	Token string
	Owner string
	Repo  string
}

type TodoistConfig struct {
	Token       string
	ProjectName string
}

type AppConfig struct {
	SyncInterval time.Duration
	Debug        bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		GitHub: GitHubConfig{
			Token: os.Getenv("GITHUB_TOKEN"),
			Owner: os.Getenv("GITHUB_OWNER"),
			Repo:  os.Getenv("GITHUB_REPO"),
		},
		Todoist: TodoistConfig{
			Token:       os.Getenv("TODOIST_TOKEN"),
			ProjectName: getEnvOrDefault("TODOIST_PROJECT_NAME", "GitHub Sync"),
		},
		App: AppConfig{
			SyncInterval: getSyncInterval(),
			Debug:        getEnvBool("DEBUG", false),
		},
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.GitHub.Token == "" {
		return fmt.Errorf("GITHUB_TOKEN je povinný")
	}
	if c.GitHub.Owner == "" {
		return fmt.Errorf("GITHUB_OWNER je povinný")
	}
	if c.GitHub.Repo == "" {
		return fmt.Errorf("GITHUB_REPO je povinný")
	}
	if c.Todoist.Token == "" {
		return fmt.Errorf("TODOIST_TOKEN je povinný")
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getSyncInterval() time.Duration {
	intervalStr := getEnvOrDefault("SYNC_INTERVAL_MINUTES", "15")
	if minutes, err := strconv.Atoi(intervalStr); err == nil {
		return time.Duration(minutes) * time.Minute
	}
	return 15 * time.Minute
}
