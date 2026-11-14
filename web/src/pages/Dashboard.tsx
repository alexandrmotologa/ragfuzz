import React, { useState } from 'react'
import type { SuiteSummary, CaseEvaluation, ChunkVulnerability, Recommendation } from '../types'
import { MetricCards } from '../components/MetricCards'
import { RadarScorecard } from '../components/RadarScorecard'
import { QueryInspector } from '../components/QueryInspector'
import { ChunkHeatmap } from '../components/ChunkHeatmap'
import { RecommendationsPanel } from '../components/RecommendationsPanel'
import { Play, RotateCw, CheckCircle2, AlertOctagon } from 'lucide-react'

interface DashboardProps {
  summary: SuiteSummary | null
  evaluations: CaseEvaluation[]
  chunks: ChunkVulnerability[]
  recommendations: Recommendation[]
  isRunning: boolean
  progress: { completed: number; total: number; currentVector?: string }
  onTriggerRun: () => void
}

export const Dashboard: React.FC<DashboardProps> = ({
  summary,
  evaluations,
  chunks,
  recommendations,
  isRunning,
  progress,
  onTriggerRun,
}) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'queries' | 'chunks' | 'mitigations'>(
    'overview'
  )

  if (!summary && !isRunning) {
    return (
      <div className="text-center py-24 bg-slate-900/40 rounded-2xl border border-slate-800 p-8 max-w-lg mx-auto">
        <AlertOctagon className="w-12 h-12 text-sky-400 mx-auto mb-4" />
        <h3 className="text-lg font-semibold text-slate-100">No Evaluation Data Loaded</h3>
        <p className="text-xs text-slate-400 mt-2 mb-6">
          Connect to a target RAG pipeline and start an adversarial fuzzing run across the 6
          targeted stress vectors.
        </p>
        <button
          onClick={onTriggerRun}
          className="inline-flex items-center gap-2 bg-sky-500 hover:bg-sky-400 text-slate-950 font-semibold text-xs px-4 py-2.5 rounded-lg transition-colors cursor-pointer"
        >
          <Play className="w-4 h-4 fill-current" /> Run Adversarial Fuzz Suite
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Live Running Progress Banner */}
      {isRunning && (
        <div className="bg-sky-950/40 border border-sky-500/40 rounded-xl p-4 flex flex-col gap-2">
          <div className="flex items-center justify-between text-xs">
            <span className="font-semibold text-sky-400 flex items-center gap-2">
              <RotateCw className="w-3.5 h-3.5 animate-spin" /> Fuzzing in progress...
            </span>
            <span className="text-slate-300 font-mono">
              {progress.completed} / {progress.total} queries (
              {((progress.completed / (progress.total || 1)) * 100).toFixed(0)}%)
            </span>
          </div>
          <div className="w-full bg-slate-900 h-2 rounded-full overflow-hidden">
            <div
              className="bg-sky-400 h-full transition-all duration-200"
              style={{
                width: `${(progress.completed / (progress.total || 1)) * 100}%`,
              }}
            />
          </div>
        </div>
      )}

      {/* Metric Cards Top Row */}
      {summary && <MetricCards summary={summary} />}

      {/* Tab Navigation */}
      <div className="flex items-center justify-between border-b border-slate-800 pb-2">
        <div className="flex gap-2 text-xs">
          <button
            onClick={() => setActiveTab('overview')}
            className={`px-3.5 py-1.5 rounded-lg font-medium transition-colors ${
              activeTab === 'overview'
                ? 'bg-sky-500/10 text-sky-400 border border-sky-500/30'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Overview & Radar
          </button>
          <button
            onClick={() => setActiveTab('queries')}
            className={`px-3.5 py-1.5 rounded-lg font-medium transition-colors ${
              activeTab === 'queries'
                ? 'bg-sky-500/10 text-sky-400 border border-sky-500/30'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Query Inspector ({evaluations.length})
          </button>
          <button
            onClick={() => setActiveTab('chunks')}
            className={`px-3.5 py-1.5 rounded-lg font-medium transition-colors ${
              activeTab === 'chunks'
                ? 'bg-sky-500/10 text-sky-400 border border-sky-500/30'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Chunk Heatmap ({chunks.length})
          </button>
          <button
            onClick={() => setActiveTab('mitigations')}
            className={`px-3.5 py-1.5 rounded-lg font-medium transition-colors ${
              activeTab === 'mitigations'
                ? 'bg-sky-500/10 text-sky-400 border border-sky-500/30'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Fix Mitigations ({recommendations.length})
          </button>
        </div>

        <button
          onClick={onTriggerRun}
          disabled={isRunning}
          className="inline-flex items-center gap-1.5 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-slate-200 text-xs px-3 py-1.5 rounded-lg border border-slate-700 transition-colors cursor-pointer"
        >
          <RotateCw className={`w-3.5 h-3.5 ${isRunning ? 'animate-spin' : ''}`} />
          {isRunning ? 'Running...' : 'Re-run Suite'}
        </button>
      </div>

      {/* Tab Panels */}
      {activeTab === 'overview' && summary && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-5">
            <div className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
              6-Axis Vulnerability Radar
            </div>
            <RadarScorecard scores={summary.vector_scores} />
          </div>

          <div className="lg:col-span-2 space-y-4">
            <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-5">
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
                Quality Gate Status & Breakdown
              </div>
              <div className="space-y-3">
                {summary.gate_breaches && summary.gate_breaches.length > 0 ? (
                  <div className="p-3 bg-rose-950/30 border border-rose-800/50 rounded-lg text-xs text-rose-300 space-y-1">
                    <div className="font-semibold text-rose-400">Breached Gate Thresholds:</div>
                    {summary.gate_breaches.map((b, idx) => (
                      <div key={idx}>&bull; {b}</div>
                    ))}
                  </div>
                ) : (
                  <div className="p-3 bg-emerald-950/30 border border-emerald-800/50 rounded-lg text-xs text-emerald-300 flex items-center gap-2">
                    <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                    <span>All production release criteria and reliability gates satisfied.</span>
                  </div>
                )}

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs pt-2">
                  <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800">
                    <div className="text-slate-400 text-[11px]">Pass Rate</div>
                    <div className="text-lg font-bold text-slate-100 mt-0.5">
                      {(summary.pass_rate * 100).toFixed(1)}%
                    </div>
                  </div>
                  <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800">
                    <div className="text-slate-400 text-[11px]">Clean Recall@K</div>
                    <div className="text-lg font-bold text-slate-100 mt-0.5">
                      {(summary.mean_recall_clean * 100).toFixed(0)}%
                    </div>
                  </div>
                  <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800">
                    <div className="text-slate-400 text-[11px]">Mean MRR</div>
                    <div className="text-lg font-bold text-slate-100 mt-0.5">
                      {summary.mean_mrr.toFixed(2)}
                    </div>
                  </div>
                  <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800">
                    <div className="text-slate-400 text-[11px]">Distractor Intrusions</div>
                    <div className="text-lg font-bold text-slate-100 mt-0.5">
                      {summary.distractor_intrusions}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Quick Failing Query Sample */}
            <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-5">
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
                Critical Failure Highlights
              </div>
              <div className="space-y-2">
                {evaluations
                  .filter((e) => !e.passed)
                  .slice(0, 3)
                  .map((e) => (
                    <div
                      key={e.test_case.id}
                      className="p-3 bg-slate-950 border border-slate-800 rounded-lg text-xs flex justify-between items-start gap-4"
                    >
                      <div>
                        <span className="font-mono text-[10px] text-sky-400 mr-2">
                          {e.test_case.id}
                        </span>
                        <span className="text-slate-200">{e.test_case.mutated_query}</span>
                        <div className="text-rose-400 text-[11px] mt-1 font-medium">
                          {e.failure_reason}
                        </div>
                      </div>
                      <button
                        onClick={() => setActiveTab('queries')}
                        className="text-sky-400 hover:text-sky-300 text-[11px] shrink-0 font-medium"
                      >
                        Inspect &rarr;
                      </button>
                    </div>
                  ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'queries' && <QueryInspector evaluations={evaluations} />}

      {activeTab === 'chunks' && <ChunkHeatmap chunks={chunks} />}

      {activeTab === 'mitigations' && (
        <RecommendationsPanel recommendations={recommendations} />
      )}
    </div>
  )
}
