import React, { useState } from 'react'
import type { CaseEvaluation, AttackVector } from '../types'
import { CheckCircle2, XCircle, ChevronDown, ChevronRight, Search, Filter } from 'lucide-react'

interface QueryInspectorProps {
  evaluations: CaseEvaluation[]
}

export const QueryInspector: React.FC<QueryInspectorProps> = ({ evaluations }) => {
  const [filterVector, setFilterVector] = useState<string>('all')
  const [filterStatus, setFilterStatus] = useState<string>('all')
  const [searchTerm, setSearchTerm] = useState<string>('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const filtered = evaluations.filter((ev) => {
    if (filterVector !== 'all' && ev.test_case.vector !== filterVector) return false
    if (filterStatus === 'passed' && !ev.passed) return false
    if (filterStatus === 'failed' && ev.passed) return false
    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      const matchQuery =
        ev.test_case.mutated_query.toLowerCase().includes(term) ||
        ev.test_case.clean_query.toLowerCase().includes(term) ||
        ev.test_case.id.toLowerCase().includes(term) ||
        ev.response.answer.toLowerCase().includes(term)
      if (!matchQuery) return false
    }
    return true
  })

  return (
    <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl overflow-hidden">
      {/* Header and Filter Controls */}
      <div className="p-4 border-b border-slate-800 flex flex-wrap gap-3 items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-2.5" />
            <input
              type="text"
              placeholder="Search query or answer..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-slate-950 border border-slate-800 rounded-lg pl-9 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-400 focus:outline-none focus:border-sky-500 w-64"
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-slate-400" />
          <select
            value={filterVector}
            onChange={(e) => setFilterVector(e.target.value)}
            className="bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-300 focus:outline-none focus:border-sky-500"
          >
            <option value="all">All Vectors</option>
            <option value="negation_inversion">Negation Inversion</option>
            <option value="semantic_distractor">Semantic Distractor</option>
            <option value="entity_version_collision">Entity / Version</option>
            <option value="orthographic_noise">Typo Noise</option>
            <option value="contradiction_trap">Contradiction Trap</option>
            <option value="out_of_domain">Out of Domain</option>
          </select>

          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
            className="bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-300 focus:outline-none focus:border-sky-500"
          >
            <option value="all">All Outcomes ({evaluations.length})</option>
            <option value="failed">
              Failures Only ({evaluations.filter((e) => !e.passed).length})
            </option>
            <option value="passed">
              Passed Only ({evaluations.filter((e) => e.passed).length})
            </option>
          </select>
        </div>
      </div>

      {/* Case List */}
      <div className="divide-y divide-slate-800/80">
        {filtered.length === 0 ? (
          <div className="p-8 text-center text-slate-400 text-xs">
            No queries match the active filter criteria.
          </div>
        ) : (
          filtered.slice(0, 100).map((ev) => {
            const isExpanded = expandedId === ev.test_case.id
            return (
              <div key={ev.test_case.id} className="transition-colors hover:bg-slate-800/30">
                <div
                  className="p-3.5 flex items-start gap-3 cursor-pointer select-none"
                  onClick={() => setExpandedId(isExpanded ? null : ev.test_case.id)}
                >
                  <button className="text-slate-400 mt-0.5">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4" />
                    ) : (
                      <ChevronRight className="w-4 h-4" />
                    )}
                  </button>

                  <div className="mt-0.5">
                    {ev.passed ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                    ) : (
                      <XCircle className="w-4 h-4 text-rose-400" />
                    )}
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-[11px] font-mono text-slate-400 font-semibold">
                        {ev.test_case.id}
                      </span>
                      <span className="text-[10px] font-mono uppercase bg-slate-800 text-sky-400 px-2 py-0.5 rounded">
                        {formatVectorTag(ev.test_case.vector)}
                      </span>
                      {ev.distractor_intruded && (
                        <span className="text-[10px] font-semibold bg-rose-950 text-rose-300 border border-rose-800 px-1.5 py-0.5 rounded">
                          DISTRACTOR INTRUDED
                        </span>
                      )}
                    </div>

                    <div className="text-sm font-medium text-slate-200">
                      {ev.test_case.mutated_query}
                    </div>

                    {!ev.passed && ev.failure_reason && (
                      <div className="text-xs text-rose-400 mt-1 font-medium">
                        Failure: {ev.failure_reason}
                      </div>
                    )}
                  </div>

                  <div className="text-right flex flex-col items-end gap-1">
                    <span
                      className={`text-xs font-semibold px-2 py-0.5 rounded ${
                        ev.passed
                          ? 'bg-emerald-950 text-emerald-300'
                          : 'bg-rose-950 text-rose-300'
                      }`}
                    >
                      {ev.passed ? 'PASSED' : 'FAILED'}
                    </span>
                    <span className="text-[11px] text-slate-400 font-mono">
                      Recall: {(ev.mutated_recall_at_k * 100).toFixed(0)}% | Grounded:{' '}
                      {(ev.groundedness * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>

                {/* Expanded Inspection Drawer */}
                {isExpanded && (
                  <div className="bg-slate-950/80 border-t border-slate-800 p-4 pl-12 text-xs space-y-4">
                    {/* Query Comparison */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="p-3 bg-slate-900 rounded-lg border border-slate-800">
                        <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide mb-1">
                          Baseline Query
                        </div>
                        <div className="text-slate-300">{ev.test_case.clean_query}</div>
                      </div>
                      <div className="p-3 bg-slate-900 rounded-lg border border-slate-800">
                        <div className="text-[11px] font-semibold text-sky-400 uppercase tracking-wide mb-1">
                          Adversarial Mutated Query
                        </div>
                        <div className="text-slate-100 font-medium">
                          {ev.test_case.mutated_query}
                        </div>
                      </div>
                    </div>

                    {/* Answer and NLI Claim Breakdown */}
                    <div className="p-3 bg-slate-900 rounded-lg border border-slate-800">
                      <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide mb-2 flex justify-between">
                        <span>Generated Response</span>
                        <span>
                          Groundedness Score: {(ev.groundedness * 100).toFixed(1)}%
                        </span>
                      </div>
                      <div className="text-slate-200 leading-relaxed bg-slate-950 p-2.5 rounded border border-slate-800 mb-3">
                        {ev.response.answer || '(Empty Answer)'}
                      </div>

                      {ev.nli && ev.nli.claims.length > 0 && (
                        <div>
                          <div className="text-[11px] font-semibold text-slate-400 mb-1.5">
                            Atomic Claim Decomposition (NLI Verification):
                          </div>
                          <div className="space-y-1.5">
                            {ev.nli.claims.map((c, cIdx) => (
                              <div
                                key={cIdx}
                                className={`p-2 rounded border flex items-start gap-2 ${
                                  c.entailed
                                    ? 'bg-emerald-950/20 border-emerald-800/40 text-emerald-300'
                                    : 'bg-rose-950/30 border-rose-800/60 text-rose-300'
                                }`}
                              >
                                {c.entailed ? (
                                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 mt-0.5 shrink-0" />
                                ) : (
                                  <XCircle className="w-3.5 h-3.5 text-rose-400 mt-0.5 shrink-0" />
                                )}
                                <div className="flex-1">
                                  <div>{c.text}</div>
                                  {!c.entailed && c.reason && (
                                    <div className="text-[10px] text-rose-400 mt-0.5 font-medium">
                                      Flag: {c.reason}
                                    </div>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>

                    {/* Retrieved Chunks */}
                    <div className="p-3 bg-slate-900 rounded-lg border border-slate-800">
                      <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide mb-2">
                        Retrieved Context Chunks ({ev.response.retrieved_chunks.length})
                      </div>
                      <div className="space-y-2">
                        {ev.response.retrieved_chunks.map((ch, idx) => {
                          const isDistractor = ev.test_case.distractor_chunk_ids?.includes(ch.id)
                          const isTarget = ev.test_case.expected_chunk_ids?.includes(ch.id)
                          return (
                            <div
                              key={idx}
                              className={`p-2.5 rounded border ${
                                isDistractor
                                  ? 'bg-rose-950/20 border-rose-700/60'
                                  : isTarget
                                  ? 'bg-sky-950/20 border-sky-700/60'
                                  : 'bg-slate-950 border-slate-800'
                              }`}
                            >
                              <div className="flex items-center justify-between text-[11px] font-mono text-slate-400 mb-1">
                                <span>
                                  Rank #{ch.rank} &bull; Chunk: {ch.id}
                                </span>
                                {isDistractor && (
                                  <span className="text-rose-400 font-semibold font-sans">
                                    Distractor Lure
                                  </span>
                                )}
                                {isTarget && (
                                  <span className="text-sky-400 font-semibold font-sans">
                                    Ground Truth
                                  </span>
                                )}
                              </div>
                              <div className="text-slate-300 font-mono text-[11px] line-clamp-3">
                                {ch.text}
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}

function formatVectorTag(v: AttackVector): string {
  switch (v) {
    case 'negation_inversion':
      return 'Negation'
    case 'semantic_distractor':
      return 'Distractor'
    case 'entity_version_collision':
      return 'Entity Swap'
    case 'orthographic_noise':
      return 'Typo Noise'
    case 'contradiction_trap':
      return 'Contradiction'
    case 'out_of_domain':
      return 'OOD Lure'
    default:
      return v
  }
}
