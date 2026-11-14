import React, { useState } from 'react'
import type { ChunkVulnerability } from '../types'
import { FileText } from 'lucide-react'

interface ChunkHeatmapProps {
  chunks: ChunkVulnerability[]
}

export const ChunkHeatmap: React.FC<ChunkHeatmapProps> = ({ chunks }) => {
  const [selectedChunk, setSelectedChunk] = useState<ChunkVulnerability | null>(
    chunks.length > 0 ? chunks[0] : null
  )

  const sorted = [...chunks].sort((a, b) => b.vulnerability_rate - a.vulnerability_rate)

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
      {/* Heatmap Grid */}
      <div className="lg:col-span-2 bg-slate-900/60 border border-slate-800/80 rounded-xl p-4">
        <div className="flex items-center justify-between mb-3">
          <div>
            <div className="text-sm font-semibold text-slate-200">
              Corpus Chunk Vulnerability Heatmap
            </div>
            <div className="text-xs text-slate-400">
              {chunks.length} chunks analyzed &bull; Sorted by failure vulnerability rate
            </div>
          </div>
          <div className="flex items-center gap-3 text-xs text-slate-400">
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-emerald-500 inline-block" /> Robust
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-amber-500 inline-block" /> Moderate
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-rose-500 inline-block" /> Vulnerable
            </span>
          </div>
        </div>

        <div className="grid grid-cols-4 sm:grid-cols-6 md:grid-cols-8 gap-2 max-h-[440px] overflow-y-auto p-1">
          {sorted.map((ch) => {
            const isSelected = selectedChunk?.chunk_id === ch.chunk_id
            let bgClass = 'bg-emerald-950/40 border-emerald-800/60 text-emerald-400'
            if (ch.vulnerability_rate > 0.4) {
              bgClass = 'bg-rose-950/60 border-rose-600 text-rose-300'
            } else if (ch.vulnerability_rate > 0.15) {
              bgClass = 'bg-amber-950/50 border-amber-700/80 text-amber-300'
            }

            return (
              <button
                key={ch.chunk_id}
                onClick={() => setSelectedChunk(ch)}
                className={`p-2 rounded-lg border text-left transition-all relative ${bgClass} ${
                  isSelected ? 'ring-2 ring-sky-400 scale-[1.02]' : 'hover:opacity-90'
                }`}
              >
                <div className="text-[10px] font-mono truncate font-semibold">
                  {ch.heading || ch.chunk_id.split('#')[0]}
                </div>
                <div className="text-xs font-bold mt-1">
                  {(ch.vulnerability_rate * 100).toFixed(0)}%
                </div>
                <div className="text-[9px] text-slate-400 mt-0.5">
                  {ch.failed_count}/{ch.total_tested}
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Selected Chunk Details */}
      <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4 flex flex-col justify-between">
        {selectedChunk ? (
          <div className="space-y-3">
            <div className="flex items-start justify-between border-b border-slate-800 pb-3">
              <div>
                <div className="text-xs font-mono text-sky-400">
                  {selectedChunk.chunk_id}
                </div>
                <div className="text-sm font-semibold text-slate-100 mt-0.5">
                  {selectedChunk.heading || 'Document Chunk'}
                </div>
                <div className="text-xs text-slate-400">{selectedChunk.doc_path}</div>
              </div>
              <div className="text-right">
                <span
                  className={`text-xs font-semibold px-2 py-0.5 rounded ${
                    selectedChunk.vulnerability_rate > 0.3
                      ? 'bg-rose-950 text-rose-300'
                      : 'bg-emerald-950 text-emerald-300'
                  }`}
                >
                  {(selectedChunk.vulnerability_rate * 100).toFixed(1)}% Fail Rate
                </span>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-2 text-center text-xs">
              <div className="bg-slate-950 p-2 rounded border border-slate-800">
                <div className="text-slate-400 text-[10px]">Queries Tested</div>
                <div className="font-semibold text-slate-200 mt-0.5">
                  {selectedChunk.total_tested}
                </div>
              </div>
              <div className="bg-slate-950 p-2 rounded border border-slate-800">
                <div className="text-slate-400 text-[10px]">Retrieval Fails</div>
                <div className="font-semibold text-rose-400 mt-0.5">
                  {selectedChunk.failed_count}
                </div>
              </div>
              <div className="bg-slate-950 p-2 rounded border border-slate-800">
                <div className="text-slate-400 text-[10px]">Distractor Hits</div>
                <div className="font-semibold text-amber-400 mt-0.5">
                  {selectedChunk.distractor_hits}
                </div>
              </div>
            </div>

            <div>
              <div className="text-xs font-semibold text-slate-400 mb-1 flex items-center gap-1.5">
                <FileText className="w-3.5 h-3.5" /> Chunk Text Content
              </div>
              <div className="bg-slate-950 border border-slate-800 rounded-lg p-3 text-xs text-slate-300 font-mono leading-relaxed max-h-56 overflow-y-auto">
                {selectedChunk.text || '(No text content)'}
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center text-slate-400 text-xs py-12">
            Select any chunk from the heatmap to inspect its vulnerability profile.
          </div>
        )}
      </div>
    </div>
  )
}
