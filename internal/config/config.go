package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)(?::-(.*?))?\}`)

// DefaultConfig returns a fully populated configuration with production-ready defaults.
func DefaultConfig() *Config {
	return &Config{
		Target: TargetConfig{
			Endpoint:        "http://localhost:8080/query",
			Method:          "POST",
			Headers:         map[string]string{"Content-Type": "application/json"},
			Timeout:         10 * time.Second,
			Concurrency:     10,
			RateLimitRPS:    50,
			RequestTemplate: `{"query":"{query}"}`,
			ResponseMapping: ResponseMapping{
				AnswerPath:    "answer",
				ChunksPath:    "chunks",
				ChunkIDPath:   "id",
				ChunkTextPath: "text",
			},
		},
		Corpus: CorpusConfig{
			Paths:          []string{"./examples/corpus"},
			ChunkSize:      800,
			ChunkOverlap:   150,
			MinChunkLength: 50,
		},
		Fuzzer: FuzzerConfig{
			TotalQueries:           100,
			EnabledVectors:         []string{},
			MaxMutationsPerChunk:   5,
			LevenshteinMaxDistance: 2,
			Seed:                   42,
		},
		Thresholds: Thresholds{
			MinGroundedness:      0.85,
			MaxRecallDegradation: 0.15,
			MinRefusalF1:         0.90,
			MaxHallucinationRate: 0.10,
		},
		Oracle: OracleConfig{
			Mode: "heuristic",
			TopK: 5,
		},
		Reporting: ReportingConfig{
			Formats:   []string{"terminal", "junit", "json", "html"},
			OutputDir: ".",
			JUnitFile: "junit.xml",
			JSONFile:  "ragfuzz-report.json",
			HTMLFile:  "report.html",
		},
		Studio: StudioConfig{
			Port:        4040,
			OpenBrowser: false,
		},
	}
}

// ExpandEnv substitutes environment variables in the format ${VAR} or ${VAR:-default}.
func ExpandEnv(input []byte) []byte {
	expanded := envVarPattern.ReplaceAllStringFunc(string(input), func(match string) string {
		submatches := envVarPattern.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			varName := submatches[1]
			defaultVal := ""
			if len(submatches) >= 3 {
				defaultVal = submatches[2]
			}
			if val, exists := os.LookupEnv(varName); exists && val != "" {
				return val
			}
			return defaultVal
		}
		return match
	})
	return []byte(expanded)
}

// LoadConfig reads a YAML configuration file from disk, applies environment variable expansion,
// and returns a validated Config instance.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", path, err)
	}

	expanded := ExpandEnv(data)

	if err := yaml.Unmarshal(expanded, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// ApplyOverrides applies CLI flag overrides to an existing Config instance.
func (c *Config) ApplyOverrides(targetEndpoint string, concurrency int, totalQueries int, failBelowGroundedness float64) {
	if strings.TrimSpace(targetEndpoint) != "" {
		c.Target.Endpoint = targetEndpoint
	}
	if concurrency > 0 {
		c.Target.Concurrency = concurrency
	}
	if totalQueries > 0 {
		c.Fuzzer.TotalQueries = totalQueries
	}
	if failBelowGroundedness > 0 {
		c.Thresholds.MinGroundedness = failBelowGroundedness
	}
}
