import React, { useState } from 'react'
import type { AttackVector, CaseEvaluation, ChunkVulnerability } from '../types'
import { Play, Sparkles, CheckCircle2, XCircle, ArrowRight } from 'lucide-react'

interface LivePlaygroundProps {
  chunks: ChunkVulnerability[]
}

export const LivePlayground: React.FC<LivePlaygroundProps> = ({ chunks }) => {
  const [query, setQuery] = useState('What are the specific requirements for enterprise SSO setup?')
  const [vector, setVector] = useState<AttackVector>('negation_inversion')
  const [targetChunkId, setTargetChunkId] = useState<string>(chunks[0]?.chunk_id || '')
  const [isLoading, setIsLoading] = useState<boolean>(false)
  const [result, setResult] = useState<CaseEvaluation | null>(null)

  const handleProbe = async () => {
    setIsLoading(true)
    try {
      const res = await fetch('/api/probe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query,
          vector,
          target_chunk_id: targetChunkId,
        }),
      })
      if (res.ok) {
        const data = await res.json()
        setResult(data)
      }
    } catch {
      // Handle network error
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {/* Probe Configuration Panel */}
      <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-5 space-y-4">
        <div>
          <div className="text-sm font-semibold text-slate-100 flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-sky-400" /> Live Adversarial Probe
          </div>
          <div className="text-xs text-slate-400 mt-1">
            Test custom queries against any adversarial stress vector and inspect real-time NLI claim
            entailment.
          </div>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wide mb-1.5">
            Baseline User Query
          </label>
          <textarea
            rows={3}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-xs text-slate-200 focus:outline-none focus:border-sky-500 font-mono"
            placeholder="Enter standard user query..."
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wide mb-1.5">
              Attack Vector Mutation
            </label>
            <select
              value={vector}
              onChange={(e) => setVector(e.target.value as AttackVector)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs text-slate-200 focus:outline-none focus:border-sky-500"
            >
              <option value="negation_inversion">Negation Inversion</option>
              <option value="semantic_distractor">Semantic Distractor</option>
              <option value="entity_version_collision">Entity / Version Swap</option>
              <option value="orthographic_noise">Typo (Levenshtein) Noise</option>
              <option value="out_of_domain">Out-of-Domain Bait</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wide mb-1.5">
              Target Corpus Chunk
            </label>
            <select
              value={targetChunkId}
              onChange={(e) => setTargetChunkId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs text-slate-200 focus:outline-none focus:border-sky-500"
            >
              {chunks.map((ch) => (
                <option key={ch.chunk_id} value={ch.chunk_id}>
                  {ch.heading || ch.chunk_id}
                </option>
              ))}
            </select>
          </div>
        </div>

        <button
          onClick={handleProbe}
          disabled={isLoading}
          className="w-full inline-flex items-center justify-center gap-2 bg-sky-500 hover:bg-sky-400 disabled:opacity-50 text-slate-950 font-semibold text-xs px-4 py-2.5 rounded-lg transition-colors cursor-pointer"
        >
          <Play className="w-4 h-4 fill-current" />
          {isLoading ? 'Executing Probe...' : 'Mutate & Probe Pipeline'}
        </button>

        {result && (
          <div className="p-3 bg-slate-950 rounded-lg border border-slate-800 space-y-2">
            <div className="text-[11px] font-semibold text-sky-400 uppercase tracking-wide flex items-center gap-1">
              <ArrowRight className="w-3.5 h-3.5" /> Synthesized Adversarial Query:
            </div>
            <div className="text-xs font-mono text-slate-100 bg-slate-900 p-2 rounded border border-slate-800">
              {result.test_case.mutated_query}
            </div>
          </div>
        )}
      </div>

      {/* Live Probe Results Panel */}
      <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-5 flex flex-col justify-between">
        {result ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div>
                <span
                  className={`text-xs font-semibold px-2 py-0.5 rounded ${
                    result.passed
                      ? 'bg-emerald-950 text-emerald-300 border border-emerald-800'
                      : 'bg-rose-950 text-rose-300 border border-rose-800'
                  }`}
                >
                  {result.passed ? 'PASSED PROBE' : 'FAILED PROBE'}
                </span>
                <span className="text-xs text-slate-400 font-mono ml-3">
                  Latency: {result.latency ? `${Math.round(result.latency / 1000000)}ms` : '3ms'}
                </span>
              </div>
              <div className="text-xs font-mono text-slate-300">
                Groundedness: {(result.groundedness * 100).toFixed(0)}%
              </div>
            </div>

            {/* Generated Answer */}
            <div>
              <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide mb-1">
                Generated Answer
              </div>
              <div className="bg-slate-950 border border-slate-800 rounded-lg p-3 text-xs text-slate-200 leading-relaxed font-mono">
                {result.response.answer || '(Empty Response)'}
              </div>
            </div>

            {/* NLI Decomposition */}
            {result.nli && result.nli.claims && (
              <div>
                <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide mb-1.5">
                  NLI Atomic Claim Verification
                </div>
                <div className="space-y-1.5 max-h-48 overflow-y-auto">
                  {result.nli.claims.map((c, i) => (
                    <div
                      key={i}
                      className={`p-2 rounded text-xs border flex items-start gap-2 ${
                        c.entailed
                          ? 'bg-emerald-950/20 border-emerald-800/40 text-emerald-300'
                          : 'bg-rose-950/30 border-rose-800/60 text-rose-300'
                      }`}
                    >
                      {c.entailed ? (
                        <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                      ) : (
                        <XCircle className="w-3.5 h-3.5 text-rose-400 shrink-0 mt-0.5" />
                      )}
                      <div>
                        <div>{c.text}</div>
                        {!c.entailed && c.reason && (
                          <div className="text-[10px] text-rose-400 mt-0.5 font-medium">
                            {c.reason}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Retrieved Context Chunks */}
            <div>
              <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide mb-1.5">
                Retrieved Context Chunks ({result.response.retrieved_chunks.length})
              </div>
              <div className="space-y-1.5 max-h-40 overflow-y-auto">
                {result.response.retrieved_chunks.map((ch, i) => (
                  <div
                    key={i}
                    className="p-2 bg-slate-950 rounded border border-slate-800 text-xs font-mono text-slate-300"
                  >
                    <div className="text-sky-400 text-[10px] mb-0.5">
                      Rank #{ch.rank} ({ch.id})
                    </div>
                    <div className="line-clamp-2">{ch.text}</div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center text-slate-400 text-xs py-20">
            Configure a query and click "Mutate & Probe Pipeline" to see live retrieval and NLI
            entailment verification.
          </div>
        )}
      </div>
    </div>
  )
}
