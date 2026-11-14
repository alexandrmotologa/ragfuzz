import React, { useEffect, useState } from 'react'
import type {
  SuiteSummary,
  CaseEvaluation,
  ChunkVulnerability,
  Recommendation,
} from './types'
import { Dashboard } from './pages/Dashboard'
import { Radio, RefreshCw, Code2 } from 'lucide-react'

export const App: React.FC = () => {
  const [summary, setSummary] = useState<SuiteSummary | null>(null)
  const [evaluations, setEvaluations] = useState<CaseEvaluation[]>([])
  const [chunks, setChunks] = useState<ChunkVulnerability[]>([])
  const [recommendations, setRecommendations] = useState<Recommendation[]>([])
  const [isRunning, setIsRunning] = useState<boolean>(false)
  const [progress, setProgress] = useState<{
    completed: number
    total: number
    currentVector?: string
  }>({ completed: 0, total: 100 })

  const loadData = async () => {
    try {
      const sumRes = await fetch('/api/summary')
      if (sumRes.ok) {
        const sumData = await sumRes.json()
        setSummary(sumData)
      }

      const casesRes = await fetch('/api/cases')
      if (casesRes.ok) {
        const casesData = await casesRes.json()
        setEvaluations(casesData)
      }

      const chunksRes = await fetch('/api/chunks')
      if (chunksRes.ok) {
        const chunksData = await chunksRes.json()
        setChunks(chunksData)
      }

      const recsRes = await fetch('/api/recommendations')
      if (recsRes.ok) {
        const recsData = await recsRes.json()
        setRecommendations(recsData)
      }
    } catch {
      // Offline or mock mode
    }
  }

  useEffect(() => {
    loadData()

    // Connect to SSE stream
    const eventSource = new EventSource('/api/stream')

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'start') {
          setIsRunning(true)
          setProgress({ completed: 0, total: data.total })
        } else if (data.type === 'progress') {
          setProgress({
            completed: data.completed,
            total: data.total,
            currentVector: data.vector,
          })
        } else if (data.type === 'complete') {
          setIsRunning(false)
          loadData()
        }
      } catch {
        // Parse error ignore
      }
    }

    eventSource.onerror = () => {
      // Will auto-reconnect
    }

    return () => {
      eventSource.close()
    }
  }, [])

  const handleTriggerRun = async () => {
    try {
      setIsRunning(true)
      await fetch('/api/run', { method: 'POST' })
    } catch {
      setIsRunning(false)
    }
  }

  return (
    <div className="min-h-screen bg-[#0b0f19] text-slate-100 flex flex-col">
      {/* Top Navigation Bar */}
      <header className="border-b border-slate-800/80 bg-[#0d121f]/80 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <span className="font-bold text-base tracking-tight text-white flex items-center gap-1.5">
                <span className="w-2.5 h-2.5 rounded-full bg-sky-400 inline-block shadow-[0_0_8px_rgba(56,189,248,0.8)]" />
                ragfuzz
              </span>
              <span className="text-[10px] font-mono font-semibold bg-slate-800 text-slate-300 border border-slate-700/80 px-1.5 py-0.5 rounded">
                v1.0.0
              </span>
            </div>
            <span className="text-slate-600 hidden sm:inline">/</span>
            <span className="text-xs text-slate-400 hidden sm:inline">
              Adversarial RAG Fuzzer & Stress Suite
            </span>
          </div>

          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-900/80 border border-slate-800 px-2.5 py-1 rounded-full">
              <Radio
                className={`w-3 h-3 ${isRunning ? 'text-amber-400 animate-pulse' : 'text-emerald-400'}`}
              />
              <span className="text-[11px] font-mono">
                {isRunning ? 'Execution Live' : 'Studio Ready'}
              </span>
            </div>

            <button
              onClick={loadData}
              title="Refresh Data"
              className="p-1.5 text-slate-400 hover:text-slate-200 transition-colors cursor-pointer"
            >
              <RefreshCw className="w-4 h-4" />
            </button>

            <a
              href="https://github.com/alexandrmotologa/ragfuzz"
              target="_blank"
              rel="noreferrer"
              className="p-1.5 text-slate-400 hover:text-white transition-colors"
            >
              <Code2 className="w-4 h-4" />
            </a>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 py-6">
        <Dashboard
          summary={summary}
          evaluations={evaluations}
          chunks={chunks}
          recommendations={recommendations}
          isRunning={isRunning}
          progress={progress}
          onTriggerRun={handleTriggerRun}
        />
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 py-4 text-center text-xs text-slate-400">
        ragfuzz &bull; Adversarial Query Fuzzer & Semantic Stress-Testing Engine for Production RAG
        Pipelines
      </footer>
    </div>
  )
}

export default App
