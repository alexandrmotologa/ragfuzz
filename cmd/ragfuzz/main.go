package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/analyzer"
	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
	"github.com/alexandrmotologa/ragfuzz/internal/reporter"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
	"github.com/alexandrmotologa/ragfuzz/internal/server"
)

const version = "1.1.0"

func printUsage() {
	fmt.Printf(`ragfuzz v%s :: Adversarial Query Fuzzer & Stress Suite for Production RAG

Usage:
  ragfuzz <command> [arguments]

Commands:
  run             Execute adversarial fuzzing suite against target RAG pipeline
  studio          Launch embedded React 19 evaluation studio on HTTP :4040
  mock-server     Start local self-contained mock RAG server for local verification
  diff            Compare two JSON reports to detect regressions between test runs
  export          Export generated test cases to benchmark datasets (ragas, huggingface, csv)
  pr-comment      Generate GitHub PR summary markdown from an evaluation report
  probe           Execute an on-the-fly single adversarial probe query
  version         Print ragfuzz version information

Run 'ragfuzz <command> --help' for details on specific command arguments.
`, version)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		runCommand(os.Args[2:])
	case "studio":
		studioCommand(os.Args[2:])
	case "mock-server":
		mockServerCommand(os.Args[2:])
	case "diff":
		diffCommand(os.Args[2:])
	case "export":
		exportCommand(os.Args[2:])
	case "pr-comment":
		prCommentCommand(os.Args[2:])
	case "probe":
		probeCommand(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("ragfuzz v%s (go/react19)\n", version)
		os.Exit(0)
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q. Run 'ragfuzz --help' for usage.\n", cmd)
		os.Exit(2)
	}
}

func runCommand(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to YAML configuration file")
	targetURL := fs.String("target", "", "Override target RAG endpoint URL")
	corpusPath := fs.String("corpus", "", "Override corpus directory path or glob")
	totalQueries := fs.Int("queries", 0, "Override total synthetic queries count")
	concurrency := fs.Int("concurrency", 0, "Override concurrency level")
	failBelow := fs.Float64("fail-below", 0, "Override minimum groundedness threshold (0.0 to 1.0)")
	preset := fs.String("preset", "", "Apply framework preset (langchain, llamaindex, openai, dspy, flowise)")
	githubSummaryPath := fs.String("github-summary", "", "Filepath to output GitHub PR summary markdown")
	launchStudio := fs.Bool("studio", false, "Launch interactive studio after running evaluation")
	fs.Parse(args)

	cfgPath := resolveConfigPath(*configPath)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to load configuration: %v\n", err)
		os.Exit(2)
	}

	// Apply CLI overrides
	cfg.ApplyOverrides(*targetURL, *concurrency, *totalQueries, *failBelow, *preset)
	if *corpusPath != "" {
		cfg.Corpus.Paths = []string{*corpusPath}
	}

	fmt.Printf("[+] Ingesting corpus from %v...\n", cfg.Corpus.Paths)
	chunks, err := corpus.IngestCorpus(&cfg.Corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Corpus ingestion failed: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("[+] Ingested %d corpus chunks successfully\n", len(chunks))

	topology := corpus.BuildTopology(chunks)
	fmt.Printf("[+] Topology graph constructed (%d high-overlap chunk pairs detected)\n", len(topology.HighOverlapPairs))

	fmt.Printf("[+] Synthesizing adversarial test cases across active vectors (target: %d)...\n", cfg.Fuzzer.TotalQueries)
	generator := fuzzer.NewGenerator(&cfg.Fuzzer, topology)
	suite, err := generator.GenerateSuite()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Fuzz suite generation failed: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("[+] Generated %d adversarial test cases\n", len(suite))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[!] Execution interrupted. Cancelling in-flight requests...")
		cancel()
	}()

	fmt.Printf("[+] Executing test cases against target %s (concurrency: %d, rate_limit: %d rps)...\n",
		cfg.Target.Endpoint, cfg.Target.Concurrency, cfg.Target.RateLimitRPS)

	pool := runner.NewWorkerPool(cfg, func(eval *oracle.CaseEvaluation, current, total int) {
		status := "PASS"
		if !eval.Passed {
			status = "FAIL"
		}
		if current%10 == 0 || current == total || !eval.Passed {
			fmt.Printf("\r    [%3d/%3d] (%s) %-26s -> %s", current, total, eval.TestCase.Vector, eval.TestCase.ID, status)
			if !eval.Passed {
				fmt.Printf(" (%s)\n", eval.FailureReason)
			}
		}
	})

	session, err := pool.Run(ctx, suite)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[-] Worker execution failed: %v\n", err)
		os.Exit(2)
	}
	fmt.Println()

	// Render Terminal Report
	reporter.RenderTerminal(os.Stdout, session)

	// Export requested formats
	outDir := cfg.Reporting.OutputDir
	if outDir == "" {
		outDir = "."
	}

	for _, fmtStr := range cfg.Reporting.Formats {
		switch strings.ToLower(fmtStr) {
		case "junit":
			path := filepath.Join(outDir, cfg.Reporting.JUnitFile)
			if err := reporter.WriteJUnitXML(path, session); err == nil {
				fmt.Printf("[+] Exported JUnit XML: %s\n", path)
			}
		case "json":
			path := filepath.Join(outDir, cfg.Reporting.JSONFile)
			if err := reporter.WriteJSONReport(path, session); err == nil {
				fmt.Printf("[+] Exported JSON Report: %s\n", path)
			}
		case "html":
			path := filepath.Join(outDir, cfg.Reporting.HTMLFile)
			if err := reporter.WriteHTMLReport(path, session); err == nil {
				fmt.Printf("[+] Exported Standalone HTML: %s\n", path)
			}
		}
	}

	if *githubSummaryPath != "" {
		if err := reporter.GenerateGitHubMarkdown(*githubSummaryPath, session); err == nil {
			fmt.Printf("[+] Exported GitHub PR Summary: %s\n", *githubSummaryPath)
		}
	}

	if *launchStudio || cfg.Studio.OpenBrowser {
		fmt.Printf("\n[+] Launching Embedded Studio on http://localhost:%d...\n", cfg.Studio.Port)
		srv := server.NewServer(cfg, chunks, topology, session)
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "[-] Studio failed: %v\n", err)
		}
	}

	// Exit Code for CI/CD Gates: 0 for pass, 1 for gate failure
	if !session.Summary.GatePassed {
		fmt.Println("[-] CI/CD quality gate threshold breached. Exiting with code 1.")
		os.Exit(1)
	}
	os.Exit(0)
}

func studioCommand(args []string) {
	fs := flag.NewFlagSet("studio", flag.ExitOnError)
	port := fs.Int("port", 4040, "HTTP port to bind studio")
	configPath := fs.String("config", "", "Path to YAML configuration file")
	fs.Parse(args)

	cfgPath := resolveConfigPath(*configPath)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}
	if *port > 0 {
		cfg.Studio.Port = *port
	}

	chunks, err := corpus.IngestCorpus(&cfg.Corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Note: Could not ingest corpus (%v). Studio will start in stand-alone mode.\n", err)
		chunks = nil
	}

	var topology *corpus.TopologyGraph
	if len(chunks) > 0 {
		topology = corpus.BuildTopology(chunks)
	}

	srv := server.NewServer(cfg, chunks, topology, nil)
	fmt.Printf("[+] ragfuzz Studio listening on http://localhost:%d\n", cfg.Studio.Port)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[-] Studio server error: %v\n", err)
		os.Exit(2)
	}
}

func mockServerCommand(args []string) {
	fs := flag.NewFlagSet("mock-server", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP port to bind mock RAG server")
	corpusPath := fs.String("corpus", "./examples/corpus", "Path to corpus directory")
	fs.Parse(args)

	cmdStr := fmt.Sprintf("go run ./examples/mock_rag_server.go -port %d -corpus %s", *port, *corpusPath)
	fmt.Printf("[+] Launch mock server with: %s\n", cmdStr)
}

func diffCommand(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: ragfuzz diff <baseline.json> <target.json>")
		os.Exit(2)
	}

	diff, err := analyzer.CompareReports(args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Diff failed: %v\n", err)
		os.Exit(2)
	}

	analyzer.RenderDiffTerminal(os.Stdout, diff)
	if diff.HasRegressions {
		os.Exit(1)
	}
	os.Exit(0)
}

func exportCommand(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to config YAML")
	format := fs.String("format", "ragas", "Target format: ragas, huggingface, or csv")
	outPath := fs.String("out", "ragfuzz_benchmark.json", "Output file path")
	count := fs.Int("count", 100, "Number of test cases to synthesize and export")
	fs.Parse(args)

	cfgPath := resolveConfigPath(*configPath)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.Fuzzer.TotalQueries = *count

	chunks, err := corpus.IngestCorpus(&cfg.Corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Corpus ingestion failed: %v\n", err)
		os.Exit(2)
	}

	topology := corpus.BuildTopology(chunks)
	generator := fuzzer.NewGenerator(&cfg.Fuzzer, topology)
	suite, err := generator.GenerateSuite()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Generation failed: %v\n", err)
		os.Exit(2)
	}

	if err := reporter.ExportSuiteCases(*outPath, *format, suite); err != nil {
		fmt.Fprintf(os.Stderr, "[-] Export failed: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("[+] Successfully exported %d benchmark cases to %s (%s format)\n", len(suite), *outPath, *format)
}

func prCommentCommand(args []string) {
	fs := flag.NewFlagSet("pr-comment", flag.ExitOnError)
	reportPath := fs.String("report", "ragfuzz-report.json", "Path to JSON evaluation report")
	outPath := fs.String("out", "github-summary.md", "Output markdown path")
	fs.Parse(args)

	data, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Could not read report file: %v\n", err)
		os.Exit(2)
	}

	type fullReportContainer struct {
		Summary     *oracle.SuiteSummary     `json:"summary"`
		Evaluations []*oracle.CaseEvaluation `json:"evaluations"`
	}

	var report fullReportContainer
	if err := json.Unmarshal(data, &report); err != nil {
		fmt.Fprintf(os.Stderr, "[-] Invalid JSON report: %v\n", err)
		os.Exit(2)
	}

	session := runner.NewSessionState(len(report.Evaluations), config.Thresholds{
		MinGroundedness:      0.85,
		MaxRecallDegradation: 0.15,
		MinRefusalF1:         0.90,
		MaxHallucinationRate: 0.10,
	})
	session.Evaluations = report.Evaluations
	session.Summary = report.Summary

	for _, ev := range report.Evaluations {
		session.VectorTotal[ev.TestCase.Vector]++
		if ev.Passed {
			session.VectorPass[ev.TestCase.Vector]++
		}
	}

	if err := reporter.GenerateGitHubMarkdown(*outPath, session); err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to generate markdown: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("[+] Generated GitHub PR Markdown summary in %s\n", *outPath)
}

func probeCommand(args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	targetURL := fs.String("target", "http://localhost:8080/query", "Target RAG endpoint")
	query := fs.String("query", "How does SSO authentication work?", "User query to test")
	vector := fs.String("vector", "negation_inversion", "Attack vector to mutate query with")
	fs.Parse(args)

	vec := fuzzer.AttackVector(*vector)
	mutated := *query
	switch vec {
	case fuzzer.VectorNegation:
		mutated = fmt.Sprintf("What components do NOT allow or exclude %s?", *query)
	case fuzzer.VectorTypoNoise:
		mutator := fuzzer.NewTypoMutator(2, time.Now().UnixNano())
		words := strings.Fields(*query)
		if len(words) > 0 {
			words[len(words)-1] = mutator.InjectTypo(words[len(words)-1])
			mutated = strings.Join(words, " ")
		}
	case fuzzer.VectorOOD:
		mutated = "How do I configure quantum key distribution modules for orbital relay?"
	}

	fmt.Printf("[+] Target: %s\n", *targetURL)
	fmt.Printf("[+] Baseline Query: %s\n", *query)
	fmt.Printf("[+] Mutated Query:  %s\n", mutated)

	cfg := config.DefaultConfig()
	cfg.Target.Endpoint = *targetURL
	client := runner.NewRAGClient(&cfg.Target)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Query(ctx, mutated)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Query failed: %v\n", err)
		os.Exit(2)
	}

	tc := &fuzzer.TestCase{
		ID:                "TC-PROBE-CLI",
		Vector:            vec,
		CleanQuery:        *query,
		MutatedQuery:      mutated,
		ExpectedBehavior:  fuzzer.BehaviorAnswer,
		GroundTruthAnswer: "Ground truth answer.",
	}
	if vec == fuzzer.VectorOOD {
		tc.ExpectedBehavior = fuzzer.BehaviorRefusal
	}

	eval := oracle.EvaluateCase(tc, resp, 5, 0.85)
	status := "PASS"
	if !eval.Passed {
		status = "FAIL"
	}

	fmt.Printf("\n--- Probe Result: %s ---\n", status)
	fmt.Printf("Answer: %s\n", resp.Answer)
	fmt.Printf("Groundedness: %.1f%%\n", eval.Groundedness*100)
	fmt.Printf("Retrieved Chunks: %d\n", len(resp.RetrievedChunks))
	if !eval.Passed {
		fmt.Printf("Failure Reason: %s\n", eval.FailureReason)
	}
}

func resolveConfigPath(custom string) string {
	if custom != "" {
		return custom
	}
	if _, err := os.Stat("ragfuzz.yaml"); err == nil {
		return "ragfuzz.yaml"
	}
	if _, err := os.Stat("examples/ragfuzz.yaml"); err == nil {
		return "examples/ragfuzz.yaml"
	}
	return ""
}
