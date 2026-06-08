package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// fileConfig represents the .mutagen.yaml config file.
type fileConfig struct {
	Packages     []string      `yaml:"packages"`
	Workers      int           `yaml:"workers"`
	Timeout      time.Duration `yaml:"timeout"`
	Threshold    float64       `yaml:"threshold"`
	Output       string        `yaml:"output"`
	HTMLOutput   string        `yaml:"html"`
	CacheDir     string        `yaml:"cache_dir"`
	NoCache      bool          `yaml:"no_cache"`
	Verbose      bool          `yaml:"verbose"`
	Exclude      []string      `yaml:"exclude"`
	Mutators     []string      `yaml:"mutators"`
	Coverprofile string        `yaml:"coverprofile"`
	DiffBase     string        `yaml:"diff"`
}

// loadConfigFile loads configuration from a YAML file.
// If path is empty, it searches for .mutagen.yaml in the current directory
// and parent directories up to the filesystem root.
func loadConfigFile(path string) fileConfig {
	if path != "" {
		cfg, err := readConfigFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read config %s: %v\n", path, err)
			return fileConfig{}
		}
		return cfg
	}

	// Search for .mutagen.yaml up the directory tree
	dir, err := os.Getwd()
	if err != nil {
		return fileConfig{}
	}

	for {
		candidate := filepath.Join(dir, ".mutagen.yaml")
		if _, err := os.Stat(candidate); err == nil {
			cfg, err := readConfigFile(candidate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read config %s: %v\n", candidate, err)
				return fileConfig{}
			}
			return cfg
		}

		// Also check .mutagen.yml
		candidate = filepath.Join(dir, ".mutagen.yml")
		if _, err := os.Stat(candidate); err == nil {
			cfg, err := readConfigFile(candidate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read config %s: %v\n", candidate, err)
				return fileConfig{}
			}
			return cfg
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return fileConfig{}
}

func readConfigFile(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, err
	}

	var raw struct {
		Packages     []string `yaml:"packages"`
		Workers      int      `yaml:"workers"`
		Timeout      string   `yaml:"timeout"`
		Threshold    float64  `yaml:"threshold"`
		Output       string   `yaml:"output"`
		HTMLOutput   string   `yaml:"html"`
		CacheDir     string   `yaml:"cache_dir"`
		NoCache      bool     `yaml:"no_cache"`
		Verbose      bool     `yaml:"verbose"`
		Exclude      []string `yaml:"exclude"`
		Mutators     []string `yaml:"mutators"`
		Coverprofile string   `yaml:"coverprofile"`
		DiffBase     string   `yaml:"diff"`
	}

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fileConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	cfg := fileConfig{
		Packages:     raw.Packages,
		Workers:      raw.Workers,
		Threshold:    raw.Threshold,
		Output:       raw.Output,
		HTMLOutput:   raw.HTMLOutput,
		CacheDir:     raw.CacheDir,
		NoCache:      raw.NoCache,
		Verbose:      raw.Verbose,
		Exclude:      raw.Exclude,
		Mutators:     raw.Mutators,
		Coverprofile: raw.Coverprofile,
		DiffBase:     raw.DiffBase,
	}

	if raw.Timeout != "" {
		// Support both "30s" and plain number (seconds)
		if d, err := time.ParseDuration(raw.Timeout); err == nil {
			cfg.Timeout = d
		} else if strings.TrimSpace(raw.Timeout) != "" {
			fmt.Fprintf(os.Stderr, "Warning: invalid timeout %q in config, using default\n", raw.Timeout)
		}
	}

	return cfg, nil
}
