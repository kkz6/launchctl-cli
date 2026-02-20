package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const projectConfigFile = ".launchctl.yml"

type ProjectConfig struct {
	Server string `yaml:"server"`
	Site   string `yaml:"site"`
}

func LoadProject() (*ProjectConfig, error) {
	path, err := findProjectConfig()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse project config: %w", err)
	}

	return &cfg, nil
}

func SaveProject(cfg *ProjectConfig) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	path := filepath.Join(cwd, projectConfigFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write project config: %w", err)
	}

	return nil
}

func findProjectConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, projectConfigFile)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return "", fmt.Errorf("no %s found (searched up to %s)", projectConfigFile, dir)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found", projectConfigFile)
		}
		dir = parent
	}
}
