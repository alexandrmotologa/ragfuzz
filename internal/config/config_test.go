package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig() should be valid, got: %v", err)
	}

	if cfg.Target.Endpoint != "http://localhost:8080/query" {
		t.Errorf("expected default endpoint, got: %s", cfg.Target.Endpoint)
	}
	if cfg.Thresholds.MinGroundedness != 0.85 {
		t.Errorf("expected 0.85 min groundedness, got: %f", cfg.Thresholds.MinGroundedness)
	}
}

func TestPresets(t *testing.T) {
	cfg := DefaultConfig()

	// Apply LangChain preset
	err := cfg.Target.ApplyPreset("langchain")
	if err != nil {
		t.Fatalf("failed to apply langchain preset: %v", err)
	}
	if cfg.Target.ResponseMapping.AnswerPath != "output.answer" {
		t.Errorf("expected output.answer, got %s", cfg.Target.ResponseMapping.AnswerPath)
	}

	// Apply LlamaIndex preset
	err = cfg.Target.ApplyPreset("llamaindex")
	if err != nil {
		t.Fatalf("failed to apply llamaindex preset: %v", err)
	}
	if cfg.Target.ResponseMapping.ChunksPath != "source_nodes" {
		t.Errorf("expected source_nodes, got %s", cfg.Target.ResponseMapping.ChunksPath)
	}

	// Invalid preset
	err = cfg.Target.ApplyPreset("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent preset")
	}
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_RAG_PORT", "9999")
	defer os.Unsetenv("TEST_RAG_PORT")

	input := []byte("endpoint: http://localhost:${TEST_RAG_PORT}/api\ntimeout: ${MISSING_TIMEOUT:-15s}")
	expanded := string(ExpandEnv(input))

	expected1 := "endpoint: http://localhost:9999/api"
	expected2 := "timeout: 15s"

	if !containsString(expanded, expected1) {
		t.Errorf("expected %q in %q", expected1, expanded)
	}
	if !containsString(expanded, expected2) {
		t.Errorf("expected %q in %q", expected2, expanded)
	}
}

func TestLoadConfigFile(t *testing.T) {
	yamlContent := `
target:
  endpoint: "http://api.local:8000/rag"
  concurrency: 25
  timeout: 5s
  preset: "dspy"
corpus:
  paths: ["./docs"]
fuzzer:
  total_queries: 500
thresholds:
  min_groundedness: 0.92
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_ragfuzz.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Target.Endpoint != "http://api.local:8000/rag" {
		t.Errorf("expected endpoint override, got: %s", cfg.Target.Endpoint)
	}
	if cfg.Target.Concurrency != 25 {
		t.Errorf("expected concurrency 25, got: %d", cfg.Target.Concurrency)
	}
	if cfg.Target.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got: %v", cfg.Target.Timeout)
	}
	if cfg.Target.ResponseMapping.ChunksPath != "context" {
		t.Errorf("expected dspy context path, got: %s", cfg.Target.ResponseMapping.ChunksPath)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && filepath.Base(s) != "" && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
