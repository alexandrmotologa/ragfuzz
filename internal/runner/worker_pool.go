package runner

import (
	"context"
	"sync"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
)

// ProgressCallback is invoked as each test case evaluation completes.
type ProgressCallback func(eval *oracle.CaseEvaluation, current, total int)

// WorkerPool manages parallel execution of adversarial test cases against the target RAG system.
type WorkerPool struct {
	client     *RAGClient
	cfg        *config.Config
	onProgress ProgressCallback
}

// NewWorkerPool creates a new concurrent execution runner.
func NewWorkerPool(cfg *config.Config, onProgress ProgressCallback) *WorkerPool {
	return &WorkerPool{
		client:     NewRAGClient(&cfg.Target),
		cfg:        cfg,
		onProgress: onProgress,
	}
}

// Run executes the complete test suite across concurrent workers and returns the final session state.
func (wp *WorkerPool) Run(ctx context.Context, suite []*fuzzer.TestCase) (*SessionState, error) {
	session := NewSessionState(len(suite), wp.cfg.Thresholds)
	if len(suite) == 0 {
		session.Finish()
		return session, nil
	}

	concurrency := wp.cfg.Target.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	if concurrency > len(suite) {
		concurrency = len(suite)
	}

	jobs := make(chan *fuzzer.TestCase, len(suite))
	for _, tc := range suite {
		jobs <- tc
	}
	close(jobs)

	var wg sync.WaitGroup
	topK := wp.cfg.Oracle.TopK
	minGroundedness := wp.cfg.Thresholds.MinGroundedness

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tc := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				resp, err := wp.client.Query(ctx, tc.MutatedQuery)
				if err != nil {
					resp = &oracle.RAGResponse{
						Error: err.Error(),
					}
				}

				eval := oracle.EvaluateCase(tc, resp, topK, minGroundedness)
				session.RecordEvaluation(eval)

				if wp.onProgress != nil {
					wp.onProgress(eval, session.Completed, session.Total)
				}
			}
		}()
	}

	wg.Wait()
	session.Finish()
	return session, nil
}
