import React from 'react'
import type { SuiteSummary } from '../types'
import { ShieldCheck, ShieldAlert, Activity, Target, Zap, AlertTriangle } from 'lucide-react'

interface MetricCardsProps {
  summary: SuiteSummary
}

export const MetricCards: React.FC<MetricCardsProps> = ({ summary }) => {
  const isGatePassed = summary.gate_passed

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-3">
      {/* CI/CD Gate Banner Card */}
      <div
        className={`col-span-1 sm:col-span-2 lg:col-span-2 rounded-xl p-4 border flex flex-col justify-between ${
          isGatePassed
            ? 'bg-emerald-950/20 border-emerald-500/30 text-emerald-300'
            : 'bg-rose-950/20 border-rose-500/30 text-rose-300'
        }`}
      >
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold tracking-wider uppercase text-slate-400">
            CI/CD Quality Gate
          </span>
          {isGatePassed ? (
            <ShieldCheck className="w-5 h-5 text-emerald-400" />
          ) : (
            <ShieldAlert className="w-5 h-5 text-rose-400" />
          )}
        </div>
        <div className="my-2">
          <div className="text-2xl font-bold tracking-tight">
            {isGatePassed ? 'GATE PASSED' : 'GATE BREACHED'}
          </div>
          <div className="text-xs text-slate-400 mt-1">
            {isGatePassed
              ? 'All 4 reliability thresholds satisfied'
              : `${summary.gate_breaches?.length ?? 1} threshold violations detected`}
          </div>
        </div>
        <div className="text-xs font-mono text-slate-400 border-t border-slate-800/80 pt-2 flex justify-between">
          <span>{summary.passed_queries} Passed</span>
          <span className={summary.failed_queries > 0 ? 'text-rose-400 font-semibold' : ''}>
            {summary.failed_queries} Failed
          </span>
          <span>{summary.total_queries} Total</span>
        </div>
      </div>

      {/* Groundedness NLI Card */}
      <div className="rounded-xl p-4 bg-slate-900/60 border border-slate-800/80 flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 text-xs">
          <span>Groundedness (NLI)</span>
          <Activity className="w-4 h-4 text-sky-400" />
        </div>
        <div className="my-1.5">
          <div className="text-2xl font-bold text-slate-100">
            {(summary.mean_groundedness * 100).toFixed(1)}%
          </div>
          <div className="text-xs text-slate-400 mt-0.5">Threshold: &ge; 85.0%</div>
        </div>
        <div className="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden">
          <div
            className={`h-full ${
              summary.mean_groundedness >= 0.85 ? 'bg-sky-400' : 'bg-rose-400'
            }`}
            style={{ width: `${Math.min(100, summary.mean_groundedness * 100)}%` }}
          />
        </div>
      </div>

      {/* Recall Drop Card */}
      <div className="rounded-xl p-4 bg-slate-900/60 border border-slate-800/80 flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 text-xs">
          <span>Recall Degradation</span>
          <Target className="w-4 h-4 text-amber-400" />
        </div>
        <div className="my-1.5">
          <div className="text-2xl font-bold text-slate-100">
            {(summary.recall_degradation * 100).toFixed(1)}%
          </div>
          <div className="text-xs text-slate-400 mt-0.5">Max drop: &le; 15.0%</div>
        </div>
        <div className="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden">
          <div
            className={`h-full ${
              summary.recall_degradation <= 0.15 ? 'bg-emerald-400' : 'bg-rose-400'
            }`}
            style={{ width: `${Math.min(100, summary.recall_degradation * 100 * 2)}%` }}
          />
        </div>
      </div>

      {/* Refusal Calibration F1 */}
      <div className="rounded-xl p-4 bg-slate-900/60 border border-slate-800/80 flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 text-xs">
          <span>Refusal F1 (OOD)</span>
          <Zap className="w-4 h-4 text-purple-400" />
        </div>
        <div className="my-1.5">
          <div className="text-2xl font-bold text-slate-100">
            {summary.refusal_f1.toFixed(2)}
          </div>
          <div className="text-xs text-slate-400 mt-0.5">
            P: {(summary.refusal_precision * 100).toFixed(0)}% | R:{' '}
            {(summary.refusal_recall * 100).toFixed(0)}%
          </div>
        </div>
        <div className="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden">
          <div
            className={`h-full ${summary.refusal_f1 >= 0.9 ? 'bg-purple-400' : 'bg-rose-400'}`}
            style={{ width: `${Math.min(100, summary.refusal_f1 * 100)}%` }}
          />
        </div>
      </div>

      {/* Distractor Intrusions */}
      <div className="rounded-xl p-4 bg-slate-900/60 border border-slate-800/80 flex flex-col justify-between">
        <div className="flex items-center justify-between text-slate-400 text-xs">
          <span>Distractor Intrusions</span>
          <AlertTriangle className="w-4 h-4 text-rose-400" />
        </div>
        <div className="my-1.5">
          <div
            className={`text-2xl font-bold ${
              summary.distractor_intrusions > 0 ? 'text-rose-400' : 'text-emerald-400'
            }`}
          >
            {summary.distractor_intrusions}
          </div>
          <div className="text-xs text-slate-400 mt-0.5">Top-K cross-talk intrusions</div>
        </div>
        <div className="text-[11px] text-slate-400">
          Latency: avg {summary.mean_latency_ms.toFixed(0)}ms (p95: {summary.p95_latency_ms.toFixed(0)}ms)
        </div>
      </div>
    </div>
  )
}
