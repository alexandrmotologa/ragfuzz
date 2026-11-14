# Adversarial Attack Vectors in ragfuzz

ragfuzz systematically tests RAG pipelines across six adversarial stress vectors. Each vector targets a specific vulnerability common in production vector retrieval and LLM generation pipelines.

---

## 1. Negation Inversion (`negation_inversion`)

### Vulnerability Mechanism
Dense embedding models (such as text-embedding-3-small, BGE, or MiniLM) represent semantic topics well, but frequently fail at syntactic negation. Queries like "Which plans support SSO?" and "Which plans do NOT support SSO?" produce high cosine similarity (> 0.92) in dense vector space. As a result, the retriever frequently returns chunks affirming SSO support instead of chunks listing excluded tiers.

### Synthesis Strategy
- Identifies affirmative assertions and lists in corpus chunks.
- Constructs queries demanding exclusion logic ("Which endpoints do NOT require API keys?", "Which pricing tiers exclude SAML SSO?").
- Verifies whether the retriever returns chunks explaining the exclusion or whether the generator hallucinates an affirmative answer.

### Recommended Mitigation
- Implement dense-sparse hybrid search with Reciprocal Rank Fusion (RRF), combining dense vectors with BM25 keyword matching.
- Add structured metadata filtering for boolean exclusion tags.

---

## 2. Semantic Distractor Injection (`semantic_distractor`)

### Vulnerability Mechanism
In enterprise documentation, distinct sections often share common vocabulary (such as "tokens", "sessions", "authentication", or "pricing"). When a user query includes keywords from Chunk B while asking a question answered by Chunk A, the retriever often returns Chunk B in rank 1, polluting the LLM context window.

### Synthesis Strategy
- Computes pairwise TF-IDF cosine similarity across corpus chunks to identify distinct chunk pairs $(C_A, C_B)$ with topical overlap between 0.30 and 0.85.
- Derives a clean question targeted at Chunk A, and injects salient entity keywords from Chunk B.
- Evaluates whether Chunk B intrudes into top-K retrieval and whether Chunk A is displaced.

### Recommended Mitigation
- Deploy a Cross-Encoder Reranker (such as bge-reranker-large or Cohere Rerank) after initial vector retrieval.
- Increase top-K retrieval depth before reranking (for example, retrieve top-25, rerank to top-5).

---

## 3. Entity and Version Collisions (`entity_version_collision`)

### Vulnerability Mechanism
Documentation frequently covers multiple product versions (for example, v1 vs v2 API specifications) or distinct user permission tiers (such as admin vs viewer). Vector models often collapse version tokens (`v1.0` vs `v2.0`), serving deprecated parameters to users inquiring about current APIs.

### Synthesis Strategy
- Extracts version strings, user privilege roles, and configuration parameters.
- Mutates query versions or swaps roles (for example, replacing `v2.0` with `v1.0` or `admin` with `viewer`).
- Evaluates whether the retriever correctly isolates version-specific chunks or serves cross-version data.

### Recommended Mitigation
- Partition documents by version using strict vector metadata filters (`filter: { version: "v2.0" }`).
- Prepend document breadcrumbs or version headers into every text chunk during ingestion.

---

## 4. Orthographic Noise (`orthographic_noise`)

### Vulnerability Mechanism
Human users frequently make typos when entering search queries into chat interfaces or search bars. Without fuzzy matching or subword tokenization, single-character errors (such as `configration` or `authentcation`) cause steep drops in retrieval Recall@K.

### Synthesis Strategy
- Injects realistic typographic errors targeting key domain terms within Levenshtein edit distance 1 to 2.
- Employs four mutation operations: adjacent letter transposition, QWERTY keyboard physical neighbor substitutions, deletions, and insertions.
- Evaluates retrieval recall degradation between clean and corrupted query pairs.

### Recommended Mitigation
- Add fuzzy lexical matching (such as BM25 with edit-distance tolerance) alongside dense embeddings.
- Implement pre-retrieval query normalization or spell correction.

---

## 5. Corpus Contradiction Traps (`contradiction_trap`)

### Vulnerability Mechanism
Real enterprise corpora inevitably contain conflicting information across outdated release notes, migrating policies, and overlapping confluence pages. When presented with contradictory chunks in context, standard LLMs tend to hallucinate a false compromise rather than explicitly reporting the discrepancy.

### Synthesis Strategy
- Discovers chunks mentioning conflicting parameters or limits.
- Generates queries that force the RAG system to address conflicting statements directly.
- Evaluates whether the pipeline flags the ambiguity or invents an ungrounded reconciliation.

### Recommended Mitigation
- Update system prompts to enforce contradiction detection: "If retrieved documents contain conflicting statements, state the discrepancy explicitly instead of reconciling."
- Add timestamp metadata and sort retrieved chunks chronologically.

---

## 6. Out-of-Domain Baits (`out_of_domain`)

### Vulnerability Mechanism
When confronted with queries about topics completely absent from the knowledge base, uncalibrated RAG pipelines frequently generate confident hallucinations instead of calibrated refusal responses.

### Synthesis Strategy
- Generates plausible enterprise inquiries that have zero grounding in the provided corpus.
- Assesses Refusal Calibration: verifies whether the model returns an explicit refusal statement or fabricates facts.
- Computes Refusal Precision, Recall, and $F_1^{\text{refusal}}$.

### Recommended Mitigation
- Implement a retrieval distance cutoff guardrail: if top-1 chunk cosine distance exceeds threshold (for example, distance > 0.45), skip generation and immediately return a calibrated refusal message.
