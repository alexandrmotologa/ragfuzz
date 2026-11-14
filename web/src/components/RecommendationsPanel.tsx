import React from 'react'
import type { Recommendation } from '../types'
import { Wrench, CheckCircle, ArrowRight } from 'lucide-react'

interface RecommendationsPanelProps {
  recommendations: Recommendation[]
}

export const RecommendationsPanel: React.FC<RecommendationsPanelProps> = ({
  recommendations,
}) => {
  if (recommendations.length === 0) {
    return (
      <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-8 text-center">
        <CheckCircle className="w-8 h-8 text-emerald-400 mx-auto mb-2" />
        <div className="text-sm font-semibold text-slate-200">
          No Critical Vulnerabilities Detected
        </div>
        <div className="text-xs text-slate-400 mt-1 max-w-md mx-auto">
          The evaluated RAG pipeline demonstrated robust resistance across all 6 adversarial
          stress vectors and satisfied all quality gate thresholds.
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {recommendations.map((rec, idx) => {
        const isCritical = rec.severity === 'CRITICAL'
        const isHigh = rec.severity === 'HIGH'
        const badgeClass = isCritical
          ? 'bg-rose-950 text-rose-300 border-rose-800'
          : isHigh
          ? 'bg-amber-950 text-amber-300 border-amber-800'
          : 'bg-sky-950 text-sky-300 border-sky-800'

        return (
          <div
            key={idx}
            className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4 flex items-start gap-4"
          >
            <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800 text-sky-400 mt-0.5 shrink-0">
              <Wrench className="w-5 h-5" />
            </div>

            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <span
                  className={`text-[10px] font-semibold tracking-wider uppercase border px-2 py-0.5 rounded-full ${badgeClass}`}
                >
                  {rec.severity} SEVERITY
                </span>
                <span className="text-xs font-mono text-slate-400">
                  Robustness: {rec.score.toFixed(1)}%
                </span>
              </div>

              <div className="text-sm font-semibold text-slate-100">{rec.title}</div>
              <div className="text-xs text-slate-300 mt-1">{rec.action_plan}</div>

              <div className="mt-3 p-3 bg-slate-950 rounded-lg border border-slate-800/90 text-xs">
                <div className="text-[11px] font-semibold text-sky-400 uppercase tracking-wide flex items-center gap-1 mb-1">
                  <ArrowRight className="w-3.5 h-3.5" /> Recommended Architectural Mitigation:
                </div>
                <div className="text-slate-200 font-mono">{rec.suggested_fix}</div>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
