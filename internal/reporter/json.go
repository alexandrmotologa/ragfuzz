package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

// FullReport represents the complete serialization envelope for test results.
type FullReport struct {
	Summary     *oracle.SuiteSummary     `json:"summary"`
	Evaluations []*oracle.CaseEvaluation `json:"evaluations"`
}

// WriteJSONReport serializes the complete evaluation results to a single JSON file.
func WriteJSONReport(filePath string, session *runner.SessionState) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	report := FullReport{
		Summary:     session.Summary,
		Evaluations: session.Evaluations,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// WriteJSONL writes evaluations line-by-line in JSONL format for streaming consumers.
func WriteJSONL(filePath string, evaluations []*oracle.CaseEvaluation) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	for _, ev := range evaluations {
		if err := encoder.Encode(ev); err != nil {
			return err
		}
	}

	return nil
}
