package reporter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
)

// ExportBenchmarkDataset serializes test cases or evaluations into external dataset formats.
func ExportBenchmarkDataset(filePath string, format string, evaluations []*oracle.CaseEvaluation) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	switch strings.ToLower(format) {
	case "ragas":
		return exportRagas(filePath, evaluations)
	case "huggingface", "hf":
		return exportHuggingFace(filePath, evaluations)
	case "csv":
		return exportCSV(filePath, evaluations)
	default:
		return fmt.Errorf("unsupported export format %q (supported: ragas, huggingface, csv)", format)
	}
}

type ragasItem struct {
	Question      string   `json:"question"`
	GroundTruth   string   `json:"ground_truth"`
	Contexts      []string `json:"contexts"`
	EvolutionType string   `json:"evolution_type"`
	Metadata      map[string]interface{} `json:"metadata"`
}

func exportRagas(path string, evaluations []*oracle.CaseEvaluation) error {
	var items []ragasItem
	for _, ev := range evaluations {
		var contexts []string
		for _, c := range ev.Response.RetrievedChunks {
			contexts = append(contexts, c.Text)
		}

		items = append(items, ragasItem{
			Question:      ev.TestCase.MutatedQuery,
			GroundTruth:   ev.TestCase.GroundTruthAnswer,
			Contexts:      contexts,
			EvolutionType: string(ev.TestCase.Vector),
			Metadata: map[string]interface{}{
				"id":                ev.TestCase.ID,
				"clean_query":       ev.TestCase.CleanQuery,
				"expected_behavior": ev.TestCase.ExpectedBehavior,
				"passed":            ev.Passed,
				"groundedness":      ev.Groundedness,
				"recall_at_k":       ev.MutatedRecallAtK,
			},
		})
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

type huggingFaceItem struct {
	ID               string `json:"id"`
	Question         string `json:"question"`
	BaselineQuestion string `json:"baseline_question"`
	GroundTruth      string `json:"ground_truth"`
	AttackVector     string `json:"attack_vector"`
	ExpectedBehavior string `json:"expected_behavior"`
	Category         string `json:"category"`
}

func exportHuggingFace(path string, evaluations []*oracle.CaseEvaluation) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	for _, ev := range evaluations {
		item := huggingFaceItem{
			ID:               ev.TestCase.ID,
			Question:         ev.TestCase.MutatedQuery,
			BaselineQuestion: ev.TestCase.CleanQuery,
			GroundTruth:      ev.TestCase.GroundTruthAnswer,
			AttackVector:     string(ev.TestCase.Vector),
			ExpectedBehavior: string(ev.TestCase.ExpectedBehavior),
			Category:         ev.TestCase.Category,
		}
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

func exportCSV(path string, evaluations []*oracle.CaseEvaluation) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"id", "vector", "mutated_query", "clean_query", "ground_truth", "expected_behavior", "passed", "groundedness", "recall_at_k"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, ev := range evaluations {
		row := []string{
			ev.TestCase.ID,
			string(ev.TestCase.Vector),
			ev.TestCase.MutatedQuery,
			ev.TestCase.CleanQuery,
			ev.TestCase.GroundTruthAnswer,
			string(ev.TestCase.ExpectedBehavior),
			fmt.Sprintf("%v", ev.Passed),
			fmt.Sprintf("%.2f", ev.Groundedness),
			fmt.Sprintf("%.2f", ev.MutatedRecallAtK),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// ExportSuiteCases exports raw test cases directly from the generator before execution.
func ExportSuiteCases(filePath string, format string, cases []*fuzzer.TestCase) error {
	var evals []*oracle.CaseEvaluation
	for _, tc := range cases {
		evals = append(evals, &oracle.CaseEvaluation{
			TestCase: tc,
			Response: &oracle.RAGResponse{},
		})
	}
	return ExportBenchmarkDataset(filePath, format, evals)
}
