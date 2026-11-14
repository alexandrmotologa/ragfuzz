package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// TargetConfig defines the target RAG HTTP endpoint and request format.
type TargetConfig struct {
	// Endpoint is the full URL of the RAG pipeline API (e.g. "http://localhost:8080/query").
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Method is the HTTP method to use (defaults to POST).
	Method string `yaml:"method" json:"method"`

	// Headers are optional custom HTTP headers sent with each request.
	Headers map[string]string `yaml:"headers" json:"headers"`

	// Timeout is the maximum duration per HTTP request.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// Concurrency is the number of concurrent worker goroutines.
	Concurrency int `yaml:"concurrency" json:"concurrency"`

	// RateLimitRPS is the maximum requests per second (0 means unlimited).
	RateLimitRPS int `yaml:"rate_limit_rps" json:"rate_limit_rps"`

	// RequestTemplate defines the JSON body template with {query} placeholder.
	// Defaults to `{"query": "{query}"}`.
	RequestTemplate string `yaml:"request_template" json:"request_template"`

	// ResponseMapping defines JSON paths to extract retrieved chunks and answer.
	ResponseMapping ResponseMapping `yaml:"response_mapping" json:"response_mapping"`
}

// ResponseMapping defines where to find retrieved context chunks and answer text in the API response.
type ResponseMapping struct {
	// AnswerPath is the JSON key or dot-path for the generated answer (e.g. "answer" or "response.text").
	AnswerPath string `yaml:"answer_path" json:"answer_path"`

	// ChunksPath is the JSON key or dot-path for retrieved chunks array (e.g. "chunks" or "retrieved_context").
	ChunksPath string `yaml:"chunks_path" json:"chunks_path"`

	// ChunkIDPath is the key within each chunk object for its identifier (e.g. "id" or "chunk_id").
	ChunkIDPath string `yaml:"chunk_id_path" json:"chunk_id_path"`

	// ChunkTextPath is the key within each chunk object for its text content (e.g. "text" or "content").
	ChunkTextPath string `yaml:"chunk_text_path" json:"chunk_text_path"`
}

// CorpusConfig defines the document collection to ingest and analyze.
type CorpusConfig struct {
	// Paths are file paths or glob patterns to documents (Markdown, TXT, JSON).
	Paths []string `yaml:"paths" json:"paths"`

	// ChunkSize is the target chunk size in characters when chunking raw documents.
	ChunkSize int `yaml:"chunk_size" json:"chunk_size"`

	// ChunkOverlap is the overlap between consecutive chunks in characters.
	ChunkOverlap int `yaml:"chunk_overlap" json:"chunk_overlap"`

	// MinChunkLength is the minimum text length for a chunk to be evaluated.
	MinChunkLength int `yaml:"min_chunk_length" json:"min_chunk_length"`
}

// FuzzerConfig controls adversarial query synthesis across the 6 attack vectors.
type FuzzerConfig struct {
	// TotalQueries is the target number of synthetic test cases to generate (e.g. 1000).
	TotalQueries int `yaml:"total_queries" json:"total_queries"`

	// EnabledVectors specifies which attack vectors to run.
	// If empty, all 6 vectors are enabled.
	EnabledVectors []string `yaml:"enabled_vectors" json:"enabled_vectors"`

	// MaxMutationsPerChunk limits how many variations are derived from one chunk.
	MaxMutationsPerChunk int `yaml:"max_mutations_per_chunk" json:"max_mutations_per_chunk"`

	// LevenshteinMaxDistance defines max edit distance for typographic noise (default 2).
	LevenshteinMaxDistance int `yaml:"levenshtein_max_distance" json:"levenshtein_max_distance"`

	// Seed allows reproducible pseudo-random mutation generation.
	Seed int64 `yaml:"seed" json:"seed"`
}

// Thresholds defines CI/CD quality gate pass/fail criteria.
type Thresholds struct {
	// MinGroundedness is the minimum acceptable groundedness score (0.0 to 1.0, e.g. 0.85).
	MinGroundedness float64 `yaml:"min_groundedness" json:"min_groundedness"`

	// MaxRecallDegradation is the max allowable drop in Recall@K between clean and mutated queries (e.g. 0.15).
	MaxRecallDegradation float64 `yaml:"max_recall_degradation" json:"max_recall_degradation"`

	// MinRefusalF1 is the minimum F1 score for out-of-domain refusal calibration (e.g. 0.90).
	MinRefusalF1 float64 `yaml:"min_refusal_f1" json:"min_refusal_f1"`

	// MaxHallucinationRate is the maximum allowable hallucination rate (e.g. 0.10).
	MaxHallucinationRate float64 `yaml:"max_hallucination_rate" json:"max_hallucination_rate"`
}

// OracleConfig configures the evaluation oracle.
type OracleConfig struct {
	// Mode specifies the verification engine: "heuristic" (local NLI, default) or "llm".
	Mode string `yaml:"mode" json:"mode"`

	// TopK is the evaluation cutoff for retrieval metrics (default 5).
	TopK int `yaml:"top_k" json:"top_k"`

	// LLMEndpoint is the base URL for LLM-as-a-judge (when Mode == "llm").
	LLMEndpoint string `yaml:"llm_endpoint" json:"llm_endpoint"`

	// LLMAPIKey is the API key for LLM-as-a-judge.
	LLMAPIKey string `yaml:"llm_api_key" json:"llm_api_key"`

	// LLMModel is the model identifier (e.g. "gpt-4o-mini", "gemini-1.5-flash").
	LLMModel string `yaml:"llm_model" json:"llm_model"`
}

// ReportingConfig defines export targets and formats.
type ReportingConfig struct {
	// Formats can include: "terminal", "junit", "json", "jsonl", "html".
	Formats []string `yaml:"formats" json:"formats"`

	// OutputDir is the directory where report artifacts are saved.
	OutputDir string `yaml:"output_dir" json:"output_dir"`

	// JUnitFile is the relative filename for JUnit XML output (e.g. "junit.xml").
	JUnitFile string `yaml:"junit_file" json:"junit_file"`

	// JSONFile is the filename for structured JSON output (e.g. "ragfuzz-report.json").
	JSONFile string `yaml:"json_file" json:"json_file"`

	// HTMLFile is the filename for standalone HTML report (e.g. "report.html").
	HTMLFile string `yaml:"html_file" json:"html_file"`
}

// StudioConfig controls the embedded web dashboard.
type StudioConfig struct {
	// Port is the HTTP port to bind for the studio interface (default 4040).
	Port int `yaml:"port" json:"port"`

	// OpenBrowser specifies whether to launch the browser on startup.
	OpenBrowser bool `yaml:"open_browser" json:"open_browser"`
}

// Config is the root configuration structure for ragfuzz.
type Config struct {
	Target     TargetConfig     `yaml:"target" json:"target"`
	Corpus     CorpusConfig     `yaml:"corpus" json:"corpus"`
	Fuzzer     FuzzerConfig     `yaml:"fuzzer" json:"fuzzer"`
	Thresholds Thresholds       `yaml:"thresholds" json:"thresholds"`
	Oracle     OracleConfig     `yaml:"oracle" json:"oracle"`
	Reporting  ReportingConfig  `yaml:"reporting" json:"reporting"`
	Studio     StudioConfig     `yaml:"studio" json:"studio"`
}

// Validate checks for configuration validity and sets sensible defaults where applicable.
func (c *Config) Validate() error {
	if c.Target.Endpoint == "" {
		return errors.New("target.endpoint is required")
	}

	if c.Target.Method == "" {
		c.Target.Method = "POST"
	}
	c.Target.Method = strings.ToUpper(c.Target.Method)

	if c.Target.Timeout <= 0 {
		c.Target.Timeout = 10 * time.Second
	}

	if c.Target.Concurrency <= 0 {
		c.Target.Concurrency = 10
	}

	if c.Target.RequestTemplate == "" {
		c.Target.RequestTemplate = `{"query":"{query}"}`
	}

	if c.Target.ResponseMapping.AnswerPath == "" {
		c.Target.ResponseMapping.AnswerPath = "answer"
	}
	if c.Target.ResponseMapping.ChunksPath == "" {
		c.Target.ResponseMapping.ChunksPath = "chunks"
	}

	if len(c.Corpus.Paths) == 0 {
		return errors.New("corpus.paths must contain at least one document or directory path")
	}

	if c.Corpus.ChunkSize <= 0 {
		c.Corpus.ChunkSize = 800
	}
	if c.Corpus.ChunkOverlap < 0 {
		c.Corpus.ChunkOverlap = 150
	}
	if c.Corpus.MinChunkLength <= 0 {
		c.Corpus.MinChunkLength = 50
	}

	if c.Fuzzer.TotalQueries <= 0 {
		c.Fuzzer.TotalQueries = 100
	}
	if c.Fuzzer.LevenshteinMaxDistance <= 0 {
		c.Fuzzer.LevenshteinMaxDistance = 2
	}

	if c.Oracle.Mode == "" {
		c.Oracle.Mode = "heuristic"
	}
	if c.Oracle.TopK <= 0 {
		c.Oracle.TopK = 5
	}

	if c.Studio.Port <= 0 {
		c.Studio.Port = 4040
	}

	if c.Reporting.OutputDir == "" {
		c.Reporting.OutputDir = "."
	}
	if c.Reporting.JUnitFile == "" {
		c.Reporting.JUnitFile = "junit.xml"
	}
	if c.Reporting.JSONFile == "" {
		c.Reporting.JSONFile = "ragfuzz-report.json"
	}
	if c.Reporting.HTMLFile == "" {
		c.Reporting.HTMLFile = "report.html"
	}

	// Validate thresholds in range [0, 1]
	if c.Thresholds.MinGroundedness < 0 || c.Thresholds.MinGroundedness > 1 {
		return fmt.Errorf("thresholds.min_groundedness must be between 0.0 and 1.0, got %f", c.Thresholds.MinGroundedness)
	}
	if c.Thresholds.MaxRecallDegradation < 0 || c.Thresholds.MaxRecallDegradation > 1 {
		return fmt.Errorf("thresholds.max_recall_degradation must be between 0.0 and 1.0, got %f", c.Thresholds.MaxRecallDegradation)
	}
	if c.Thresholds.MinRefusalF1 < 0 || c.Thresholds.MinRefusalF1 > 1 {
		return fmt.Errorf("thresholds.min_refusal_f1 must be between 0.0 and 1.0, got %f", c.Thresholds.MinRefusalF1)
	}

	return nil
}
