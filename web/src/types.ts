export type AttackVector =
  | 'negation_inversion'
  | 'semantic_distractor'
  | 'entity_version_collision'
  | 'orthographic_noise'
  | 'contradiction_trap'
  | 'out_of_domain'

export interface TestCase {
  id: string
  vector: AttackVector
  clean_query: string
  mutated_query: string
  target_chunk_id?: string
  expected_chunk_ids?: string[]
  distractor_chunk_ids?: string[]
  ground_truth_answer?: string
  expected_behavior: 'answer' | 'refusal' | 'reconciliation'
  category?: string
  metadata?: Record<string, string>
}

export interface RetrievedChunk {
  id: string
  text: string
  rank: number
}

export interface RAGResponse {
  answer: string
  retrieved_chunks: RetrievedChunk[]
  latency: number
  status_code: number
  error?: string
}

export interface Claim {
  text: string
  keywords: string[]
  numbers: string[]
  is_negated: boolean
  entailed: boolean
  reason?: string
}

export interface NLIResult {
  total_claims: number
  entailed_claims: number
  groundedness: number
  is_refusal: boolean
  claims: Claim[]
  hallucinated_text?: string
}

export interface CaseEvaluation {
  test_case: TestCase
  response: RAGResponse
  clean_recall_at_k: number
  mutated_recall_at_k: number
  recall_delta: number
  mrr: number
  groundedness: number
  hallucination_rate: number
  distractor_intruded: boolean
  is_refusal: boolean
  passed: boolean
  failure_reason?: string
  nli?: NLIResult
  latency: number
}

export interface SuiteSummary {
  total_queries: number
  passed_queries: number
  failed_queries: number
  pass_rate: number
  mean_groundedness: number
  mean_recall_clean: number
  mean_recall_mutated: number
  recall_degradation: number
  mean_mrr: number
  hallucination_rate: number
  refusal_precision: number
  refusal_recall: number
  refusal_f1: number
  distractor_intrusions: number
  mean_latency_ms: number
  p95_latency_ms: number
  vector_scores: Record<AttackVector, number>
  gate_passed: boolean
  gate_breaches?: string[]
  executed_at: string
  duration_ms: number
}

export interface ChunkVulnerability {
  chunk_id: string
  doc_path: string
  heading: string
  total_tested: number
  failed_count: number
  distractor_hits: number
  vulnerability_rate: number
  text: string
}

export interface Recommendation {
  vector: AttackVector
  score: number
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM'
  title: string
  action_plan: string
  suggested_fix: string
}
