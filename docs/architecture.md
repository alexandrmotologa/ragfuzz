# ragfuzz System Architecture

ragfuzz is a single-binary adversarial query fuzzer and semantic stress-testing engine designed for production Retrieval-Augmented Generation (RAG) pipelines. It combines a high-throughput Go engine (< 20MB RAM footprint) with an embedded React 19 evaluation studio.

```
+---------------------------------------------------------------------------------+
|                        Corpus Ingestion & Topology Graph                        |
|  (Markdown, JSON, Plaintext -> Sliding-Window Chunks -> BM25 Inverted Index)   |
+---------------------------------------------------------------------------------+
                                       |
                                       v
+---------------------------------------------------------------------------------+
|                         6-Vector Adversarial Mutator                            |
|  - Negation Inversion             - Orthographic Levenshtein Noise              |
|  - Semantic Distractor Injection  - Knowledge Base Contradiction Traps          |
|  - Entity & Version Swaps         - Out-of-Domain Refusal Baits                 |
+---------------------------------------------------------------------------------+
                                       |
                                       v
+---------------------------------------------------------------------------------+
|                     Concurrent Rate-Limited Runner Pool                         |
|  (Token-Bucket RPS Throttling, Exponential Jitter Backoff, Generic JSON Schema)  |
+---------------------------------------------------------------------------------+
                                       |
                        +--------------+--------------+
                        | Query                       | Answer + Retrieved Chunks
                        v                             v
+-------------------------------+             +-----------------------------------+
|      Target RAG Pipeline      |             |       Dual Evaluation Oracle      |
|  (HTTP API / LangChain / vLLM)|             |  - Retrieval: Recall@K, MRR       |
+-------------------------------+             |  - Generation: Groundedness (NLI) |
                                              |  - Refusal Calibration F1         |
                                              +-----------------------------------+
                                                              |
                                                              v
+---------------------------------------------------------------------------------+
|                             Reporting & Integration                             |
|  - Embedded React 19 Studio (:4040)                                             |
|  - Standalone Single-File HTML Report (Inline SVG Radar)                        |
|  - JUnit XML & JSONL for CI/CD Gates (Exit Code 0 on pass, 1 on breach)         |
|  - Baseline Regression Diff Analyzer (ragfuzz diff)                             |
+---------------------------------------------------------------------------------+
```

---

## 1. Corpus Ingestion and Knowledge Graph Topology

The corpus layer (`internal/corpus`) ingests Markdown, plain text, and JSON documents.

### Heading-Aware Chunking
For Markdown documents, the chunker respects document hierarchy by splitting along `#`, `##`, and `###` headers. If a section exceeds the configured `chunk_size`, a sliding window algorithm splits the section while preserving sentence boundaries and applying `chunk_overlap`.

### Fast Entity Extraction
Rule-based entity recognition isolates:
- Versions (such as `v1.0`, `v2.4`, `version 3.0`)
- Roles (such as `admin`, `viewer`, `operator`, `billing_admin`)
- Parameters and technical identifiers (`rate_limit_rps`, `auth_token`)
- Numeric quotas and thresholds (`10MB`, `500ms`, `100rps`)

### BM25 Topology and Topic Clustering
The engine computes Robertson-Sparck Jones BM25 term weights across all chunks. Additionally, pairwise TF-IDF cosine similarities are calculated across the corpus to construct a similarity matrix:
- Pairs with high topical overlap (similarity between 0.30 and 0.85) are identified as candidates for cross-chunk distractor injection.
- Chunks sharing identical entities with conflicting metrics are flagged for contradiction traps.

---

## 2. Adversarial Mutator Engine

The fuzzer engine (`internal/fuzzer`) generates synthetic test cases across six targeted attack vectors:

1. **Negation Inversion (`negation_inversion`)**: Converts affirmative inquiries into negative/exclusionary clauses (for example, "Which tiers do NOT support SAML?"). This evaluates whether vector embeddings falsely cluster negative queries with positive assertions.
2. **Semantic Distractor Injection (`semantic_distractor`)**: Injects salient keywords from adjacent chunks into inquiries targeting a different chunk, provoking retriever cross-talk.
3. **Entity and Version Collisions (`entity_version_collision`)**: Swaps version strings and user privilege roles to test whether retrievers collapse version boundaries.
4. **Orthographic Noise (`orthographic_noise`)**: Introduces controlled Levenshtein distance 1-2 edits and QWERTY adjacency errors to assess retriever resilience under human typos.
5. **Corpus Contradiction Traps (`contradiction_trap`)**: Generates queries targeting conflicting corpus statements to test whether the pipeline acknowledges discrepancies or fabricates a compromise.
6. **Out-of-Domain Baits (`out_of_domain`)**: Enterprise-plausible questions that have zero grounding in the knowledge base, evaluating refusal calibration.

---

## 3. High-Throughput Concurrent Runner

The runner (`internal/runner`) dispatches queries across a configurable pool of worker goroutines:
- **Rate Limiting**: Employs a token-bucket rate limiter to avoid overwhelming target endpoints.
- **Retry with Jitter**: Automatically retries requests receiving HTTP 429, 502, 503, or 504 using exponential backoff with randomized jitter.
- **Template Substitution**: Maps test queries into custom JSON request templates (`{"query": "{query}"}`).

---

## 4. Dual Verification Oracle

The oracle (`internal/oracle`) evaluates both retrieval and generation stages:

### Retrieval Stage Metrics
- **Recall@K**: Evaluates the fraction of expected supporting chunks present in the top-K retrieved context.
- **Mean Reciprocal Rank (MRR)**: Evaluates the rank position of the first ground-truth chunk.
- **Distractor Intrusion Rate**: Flags instances where distractor chunks were mistakenly retrieved into the top-K context.

### Generation Stage Metrics
- **Groundedness Score**: An in-memory, deterministic Natural Language Inference (NLI) engine decomposes generated responses into atomic claims, checking directional entailment, number alignment, and negation polarity against retrieved chunks.
- **Refusal Calibration ($F_1^{\text{refusal}}$)**: Evaluates precision and recall in rejecting unanswerable out-of-domain queries while answering valid in-domain queries.

---

## 5. Output and Studio Architecture

- **Terminal Scorecard**: Rich ANSI table summary displaying metric thresholds, 6-axis radar breakdown, and gate status.
- **JUnit XML**: Machine-readable test suite export compatible with GitHub Actions, GitLab CI, and Jenkins.
- **Single-File HTML**: Self-contained report bundling interactive SVG radar charts and searchable query diff viewers with zero external CDN dependencies.
- **Embedded React 19 Studio**: Static bundle compiled into the Go binary via `//go:embed`, providing live SSE streaming and interactive query debugging on port 4040.
