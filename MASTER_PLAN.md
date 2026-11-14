# ragfuzz: Architecture and Engineering Master Plan
> Automated Adversarial Query Fuzzer, Semantic Stress-Testing Suite, and Groundedness Benchmark for Production RAG Pipelines
> Target: High-Performance Single-Binary Go Engine (< 20MB RAM) with Embedded React 19 Evaluation Studio & CI/CD Quality Gates

---

## 1. Problem Statement and Motivation

### 1.1 The Fragility of Production RAG Systems
Retrieval-Augmented Generation (RAG) is the dominant architecture for grounding LLMs in private enterprise data. However, production RAG pipelines suffer from silent, catastrophic failure modes that standard evaluations fail to uncover:
1. **Semantic Distractor Trapping**: Queries that share high cosine similarity with irrelevant chunks cause the retriever to crowd out the true supporting evidence.
2. **Polarity and Negation Blindness**: Embedding models frequently map affirmative and negative statements into adjacent vector spaces, causing retrievers to return evidence directly contradicting the user's inquiry (e.g., "Which tiers do NOT support SSO?").
3. **Temporal and Version Collisions**: When a corpus contains documentation across multiple product versions (e.g., v1 vs. v2 API specs), retrievers collapse temporal distinctions and feed deprecated parameters into the generation prompt.
4. **Knowledge Base Internal Contradictions**: Corpora inevitably contain conflicting data across outdated PDFs, confluence pages, and release notes. In the presence of contradictory retrieved chunks, LLMs hallucinate reconciliations rather than flagging discrepancies.
5. **Lack of Refusal Calibration**: When confronted with queries outside the knowledge base, production pipelines frequently generate confident hallucinations instead of calibrated "I cannot find this information in the provided documentation" responses.

### 1.2 Limitations of Existing Evaluation Tools
Conventional evaluation frameworks (such as Ragas, TruLens, or Giskard) exhibit major operational bottlenecks:
- **Passive Retrospective Analysis**: They evaluate queries already logged from production users, identifying vulnerabilities only after customers have experienced hallucinations.
- **Heavy Python Runtimes**: Requiring complex Python virtual environments, PyTorch dependencies, and gigabytes of memory prevents seamless integration into lightweight CI/CD build runners.
- **Synthetic Query Homogeneity**: Standard query generation simply prompts an LLM to "generate 5 questions for this text", creating easy, non-adversarial questions that achieve artificially high (>95%) baseline benchmark scores.

### 1.3 The ragfuzz Solution
`ragfuzz` is an autonomous, high-speed fuzzing and stress-testing harness written in Go. It operates directly against vector databases or raw document collections, generating hundreds to thousands of synthetic adversarial test cases across six targeted stress vectors:
- **Automated Corpus Ingestion**: Ingests Markdown, PDF, text, and JSON documents, or connects to live vector stores via standardized protocols.
- **Six-Vector Mutation Engine**: Generates semantic distractors, negation inversions, entity swaps, typographic noise, cross-chunk contradiction traps, and out-of-domain unanswerable lures.
- **Dual Verification Oracle**: Evaluates both the Retrieval stage (Recall@K, Mean Reciprocal Rank, Context Relevance) and the Generation stage (Faithfulness, Groundedness, Hallucination Rate, Refusal Calibration).
- **CI/CD Quality Gate**: Fails continuous integration pipelines with non-zero exit codes if regression thresholds are breached (e.g., Groundedness < 0.85 or Out-of-Domain Refusal < 0.95).
- **Embedded Visual Studio**: Bundles an interactive React 19 dashboard inside the single binary to visualize error heatmaps, chunk vulnerability clusters, and failure replays.

---

## 2. High-Level System Architecture

```
       +-------------------------------------------------------------+
       |               Corpus Source / Knowledge Base                |
       |  (Raw Docs: MD/PDF/JSON  OR  Vector Store: Qdrant/pgvector) |
       +-------------------------------------------------------------+
                                      |
                                      v
       +-------------------------------------------------------------+
       |                     ragfuzz Core Engine                     |
       |                   (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | Document Ingestion  |        | Corpus Graph & Cluster |  |
       |  | & Chunk Topology    |------->| Entity Extraction      |  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |           Adversarial Mutation Generator              |  |
       |  |  [Negation] [Distractor] [Entity Swap] [Typo Noise]   |  |
       |  |  [Contradiction Trap]    [Out-of-Domain Injection]    |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |               Fuzz Execution Runner                   |  |
       |  |  Concurrent Workers (Rate-Limited, Backoff-Tuned)     |  |
       |  +-------------------------------------------------------+  |
       |              |                               |              |
       +--------------|-------------------------------|--------------+
                      | Query                          | Context + Ans
                      v                                v
       +----------------------------+   +----------------------------+
       | Target RAG HTTP Endpoint   |   |   Dual Evaluation Oracle   |
       | (LangChain/LlamaIndex/API) |   | (Retrieval & Groundedness) |
       +----------------------------+   +----------------------------+
                                                       |
                                                       v
       +-------------------------------------------------------------+
       |                     Output & Integration                    |
       |  1. Embedded React 19 Studio (Local HTTP :4040)             |
       |  2. Standalone HTML Artifact (report.html)                  |
       |  3. JSONL / JUnit XML for CI/CD Gates (exit code 0 / 1)     |
       +-------------------------------------------------------------+
```

---

## 3. The 6-Vector Adversarial Mutation Engine

```
+-----------------------------------+-----------------------------------------------------------------+
| Mutation Strategy                 | Mechanism & Adversarial Goal                                    |
+-----------------------------------+-----------------------------------------------------------------+
| 1. Negation Inversion             | Synthesizes queries requiring exclusion logic. Tests if vector  |
|                                   | space mistakenly clusters negative requests with positive facts.|
| 2. Semantic Distractor Injection  | Generates queries containing terms from Chunk A whose true      |
|                                   | answer is located in Chunk B, provoking retriever cross-talk.   |
| 3. Entity & Temporal Swaps        | Swaps version flags, product names, or dates with subtle        |
|                                   | adjacent entities to exploit low-dimensional vector collapse.   |
| 4. Typographic & Phonetic Noise   | Introduces controlled Levenshtein edits and QWERTY keyboard     |
|                                   | distance misspellings to test embedding robustness under noise. |
| 5. Contradiction Traps            | Identifies conflicting statements across chunks; forces RAG     |
|                                   | pipeline to reconcile or highlight ambiguity without lying.    |
| 6. Out-of-Domain (OOD) Lures      | Plausible enterprise queries with zero grounding in corpus.     |
|                                   | Assesses refusal calibration (must reply with refusal message). |
+-----------------------------------+-----------------------------------------------------------------+
```

---

## 4. Mathematical Evaluation Metrics

### 4.1 Groundedness (Faithfulness) Score
For an answer $A$ composed of individual semantic claims $c_1, c_2, \dots, c_m$ and retrieved context set $C = \{k_1, k_2, \dots, k_n\}$:

$$\text{Groundedness}(A, C) = \frac{\sum_{i=1}^{m} \mathbb{I}(c_i \text{ is strictly entailed by } C)}{m}$$

Where $\mathbb{I}$ is the indicator function evaluated via local deterministic NLI (Natural Language Inference) models or calibrated LLM-as-a-judge.

### 4.2 Retrieval Recall@K Under Adversarial Perturbation
Given the set of golden supporting chunks $G_q$ for query $q$ and top-$K$ retrieved chunks $R_{q,K}$:

$$\text{Recall@K}(q) = \frac{|R_{q,K} \cap G_q|}{|G_q|}$$

$$\Delta \text{Recall}_{\text{stress}} = \frac{1}{|Q|} \sum_{q \in Q} \left( \text{Recall@K}(q_{\text{clean}}) - \text{Recall@K}(q_{\text{mutated}}) \right)$$

### 4.3 Out-of-Domain Refusal Calibration ($F_1^{\text{refusal}}$)
Measures the balance between correctly refusing out-of-domain queries ($Q_{\text{ood}}$) and correctly answering in-domain queries ($Q_{\text{id}}$):

$$P_{\text{refusal}} = \frac{\text{True Refusals}}{\text{True Refusals} + \text{False Refusals}}, \quad R_{\text{refusal}} = \frac{\text{True Refusals}}{|Q_{\text{ood}}|}$$

$$F_1^{\text{refusal}} = 2 \cdot \frac{P_{\text{refusal}} \cdot R_{\text{refusal}}}{P_{\text{refusal}} + R_{\text{refusal}}}$$

---

## 5. Repository Structure

```
ragfuzz/
├── cmd/
│   └── ragfuzz/
│       └── main.go                 # CLI entry point, flag parsing, signal handling
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration loader and environment overrides
│   │   └── schema.go               # Thresholds, target endpoints, vector store config
│   ├── corpus/
│   │   ├── ingestion.go            # Markdown, text, PDF, and JSON chunk loaders
│   │   ├── topology.go             # Chunk graph construction and similarity clustering
│   │   └── entity.go               # Fast entity and keyword extractor
│   ├── fuzzer/
│   │   ├── generator.go            # Adversarial test case synthesizer orchestrator
│   │   ├── negation.go             # Negation inversion mutator
│   │   ├── distractor.go           # Cross-chunk semantic distractor generator
│   │   ├── entity_swap.go          # Entity and temporal attribute mutator
│   │   ├── typo.go                 # Levenshtein and keyboard walk typo injector
│   │   ├── contradiction.go        # Cross-chunk conflicting statements detector
│   │   └── ood.go                  # Out-of-domain unanswerable query generator
│   ├── runner/
│   │   ├── client.go               # HTTP client with rate-limiting and exponential backoff
│   │   ├── worker_pool.go          # Concurrent query execution pipeline
│   │   └── session.go              # Test run state management and checkpointing
│   ├── oracle/
│   │   ├── metrics.go              # Recall@K, MRR, Groundedness, and Hallucination calculator
│   │   ├── nli.go                  # Natural Language Inference verification engine
│   │   └── judge.go                # Calibrated LLM-as-a-judge evaluation interface
│   ├── reporter/
│   │   ├── terminal.go             # Rich CLI tables, progress bars, and colored summary
│   │   ├── junit.go                # JUnit XML output for CI/CD pipeline integration
│   │   ├── json.go                 # JSONL streaming and summary export
│   │   └── html.go                 # Standalone single-file HTML report generator
│   └── server/
│       ├── api.go                  # Local REST API for interactive inspection
│       ├── embed.go                # Go 1.16+ embed filesystem for React static assets
│       └── sse.go                  # Server-Sent Events stream for live fuzzing progress
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── RadarScorecard.tsx  # 6-axis vulnerability radar chart
│   │   │   ├── ChunkHeatmap.tsx    # Interactive corpus chunk vulnerability map
│   │   │   ├── QueryInspector.tsx  # Failing query deep-dive with diff viewer
│   │   │   ├── MetricCards.tsx     # Groundedness, Recall, Hallucination, Refusal metrics
│   │   │   └── LiveConsole.tsx     # Real-time execution stream
│   │   └── pages/
│   │       └── Dashboard.tsx       # Main evaluation view
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── ragfuzz.yaml                # Standard enterprise configuration sample
│   ├── corpus/                     # Sample enterprise documentation chunks
│   │   ├── billing_policy.md
│   │   ├── sso_setup.md
│   │   └── api_v2_reference.md
│   └── mock_rag_server.go          # Self-contained mock RAG HTTP server for testing
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 6. Implementation Roadmap

### Phase 1: Go CLI Foundation and Configuration Engine
- Initialize Go module with minimal dependencies.
- Build `internal/config` supporting YAML parsing, CLI flags (`--config`, `--target`, `--workers`, `--fail-below`), and environment variable interpolation.
- Implement `examples/mock_rag_server.go` to provide a reproducible local target returning mocked retrieved context and answers.

### Phase 2: Corpus Ingestion and Knowledge Graph Clustering
- Implement document parsers for Markdown, JSON, and raw text files with customizable chunking boundaries.
- Build chunk topology analyzer in `internal/corpus/topology.go` computing pairwise TF-IDF / BM25 or embedding cosine similarities to detect overlapping topics.
- Extract high-frequency entities, dates, and domain concepts.

### Phase 3: The Six Adversarial Mutators
- Implement `negation.go` to construct queries inverted with exclusionary clauses.
- Implement `distractor.go` to inject keywords from high-similarity adjacent chunks into adversarial queries.
- Implement `entity_swap.go` and `typo.go` for syntactic and semantic noise injection.
- Implement `contradiction.go` to identify conflicting statements across corpus chunks and generate probing queries.
- Implement `ood.go` to generate enterprise-relevant inquiries absent from the corpus for refusal calibration.

### Phase 4: Concurrent Execution Runner and Oracle
- Build worker pool managing concurrent requests against the target RAG HTTP endpoint with adjustable RPS and backoff.
- Implement `internal/oracle` computing Recall@K, Groundedness Score, and Refusal Calibration $F_1$.
- Implement deterministic heuristics and LLM judge fallback with position-bias mitigation.

### Phase 5: CI/CD Quality Gates and Reporting
- Implement JUnit XML (`internal/reporter/junit.go`) and JSONL export.
- Build CLI terminal reporting with formatted summary tables and explicit pass/fail exit code status (`0` for success, `1` for threshold breach).
- Implement single-file standalone HTML report generator bundling inline SVG visualizations.

### Phase 6: Embedded React 19 Evaluation Studio
- Develop interactive web interface in `web/` using React 19, Tailwind CSS, and Lucide icons.
- Build 6-axis vulnerability radar chart, corpus chunk vulnerability heatmap, and query diff inspector.
- Embed frontend static assets into the Go binary using `//go:embed` and serve locally on `:4040`.

### Phase 7: Verification, Benchmarks, and Documentation
- Write comprehensive unit tests for all six mutators, scoring metrics, and HTTP runner.
- Conduct performance benchmarks verifying memory footprint under 20MB RAM and fuzzing throughput exceeding 50 queries/sec against local mock servers.
- Author humanized README with setup guides, CI/CD GitHub Actions workflows, and architecture diagrams.
