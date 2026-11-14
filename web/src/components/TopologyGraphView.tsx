import React, { useEffect, useState } from 'react'
import type { GraphData, GraphNode } from '../types'
import { Network, AlertTriangle } from 'lucide-react'

export const TopologyGraphView: React.FC = () => {
  const [data, setData] = useState<GraphData | null>(null)
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null)

  useEffect(() => {
    fetch('/api/graph')
      .then((r) => (r.ok ? r.json() : null))
      .then((g) => {
        if (g) setData(g)
      })
      .catch(() => {})
  }, [])

  if (!data || data.nodes.length === 0) {
    return (
      <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-8 text-center text-slate-400 text-xs">
        No topology graph data available.
      </div>
    )
  }

  // Calculate circular layout positions for nodes
  const cx = 320
  const cy = 240
  const radius = 170
  const total = data.nodes.length

  const nodePositions = new Map<string, { x: number; y: number }>()
  data.nodes.forEach((n, i) => {
    const angle = i * ((2 * Math.PI) / total) - Math.PI / 2
    const x = cx + radius * Math.cos(angle)
    const y = cy + radius * Math.sin(angle)
    nodePositions.set(n.id, { x, y })
  })

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      {/* 2D Graph Canvas */}
      <div className="lg:col-span-2 bg-slate-900/60 border border-slate-800/80 rounded-xl p-4 flex flex-col items-center">
        <div className="w-full flex items-center justify-between mb-2">
          <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
            <Network className="w-4 h-4 text-sky-400" /> Corpus Chunk Topology & Distractor Links
          </div>
          <div className="text-[11px] text-slate-400">
            {data.nodes.length} nodes &bull; {data.edges.length} topical similarity edges
          </div>
        </div>

        <svg width="640" height="480" viewBox="0 0 640 480" className="overflow-visible select-none">
          {/* Similarity Edges */}
          {data.edges.map((e, idx) => {
            const p1 = nodePositions.get(e.source)
            const p2 = nodePositions.get(e.target)
            if (!p1 || !p2) return null

            const isDistractor = e.is_distractor || e.similarity > 0.45
            const strokeColor = isDistractor ? 'rgba(239, 68, 68, 0.6)' : 'rgba(56, 189, 248, 0.3)'
            const strokeDash = isDistractor ? '3,3' : 'none'
            const strokeWidth = Math.max(1, e.similarity * 3)

            return (
              <line
                key={idx}
                x1={p1.x}
                y1={p1.y}
                x2={p2.x}
                y2={p2.y}
                stroke={strokeColor}
                strokeWidth={strokeWidth}
                strokeDasharray={strokeDash}
              />
            )
          })}

          {/* Chunk Nodes */}
          {data.nodes.map((n) => {
            const pos = nodePositions.get(n.id)
            if (!pos) return null

            const isSelected = selectedNode?.id === n.id
            const nodeFill = isSelected ? '#38bdf8' : '#1e293b'
            const strokeColor = isSelected ? '#ffffff' : '#334155'

            return (
              <g
                key={n.id}
                onClick={() => setSelectedNode(n)}
                className="cursor-pointer transition-transform hover:scale-110"
              >
                <circle
                  cx={pos.x}
                  cy={pos.y}
                  r={isSelected ? 12 : 9}
                  fill={nodeFill}
                  stroke={strokeColor}
                  strokeWidth="2"
                />
                <text
                  x={pos.x}
                  y={pos.y + 18}
                  fill="#94a3b8"
                  fontSize="10"
                  textAnchor="middle"
                  className="font-mono font-medium pointer-events-none"
                >
                  {n.heading ? n.heading.slice(0, 16) : n.id.slice(0, 10)}
                </text>
              </g>
            )
          })}
        </svg>

        <div className="flex items-center gap-4 text-xs text-slate-400 mt-2">
          <span className="flex items-center gap-1.5">
            <span className="w-3 h-0.5 bg-sky-400 inline-block" /> Topical Overlap Edge
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-3 h-0.5 border-b-2 border-dashed border-rose-500 inline-block" />{' '}
            Distractor Cross-Talk Path
          </span>
        </div>
      </div>

      {/* Selected Node Details & Connected Edges */}
      <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-5 flex flex-col justify-between">
        {selectedNode ? (
          <div className="space-y-4">
            <div className="border-b border-slate-800 pb-3">
              <div className="text-[11px] font-mono text-sky-400">{selectedNode.id}</div>
              <div className="text-sm font-semibold text-slate-100 mt-0.5">
                {selectedNode.heading || 'Document Chunk'}
              </div>
              <div className="text-xs text-slate-400">{selectedNode.doc_path}</div>
            </div>

            <div>
              <div className="text-xs font-semibold text-slate-300 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
                Interfering Distractor Neighbors:
              </div>
              <div className="space-y-2 max-h-64 overflow-y-auto">
                {data.edges
                  .filter((e) => e.source === selectedNode.id || e.target === selectedNode.id)
                  .map((e, idx) => {
                    const otherId = e.source === selectedNode.id ? e.target : e.source
                    return (
                      <div
                        key={idx}
                        className="p-2.5 bg-slate-950 rounded-lg border border-slate-800 text-xs"
                      >
                        <div className="flex justify-between items-center text-[11px] mb-1">
                          <span className="font-mono text-slate-300">{otherId}</span>
                          <span className="font-semibold text-sky-400 font-mono">
                            {(e.similarity * 100).toFixed(0)}% Sim
                          </span>
                        </div>
                        {e.shared_keys && e.shared_keys.length > 0 && (
                          <div className="text-[10px] text-slate-400">
                            Shared terms: {e.shared_keys.slice(0, 4).join(', ')}
                          </div>
                        )}
                      </div>
                    )
                  })}
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center text-slate-400 text-xs py-24">
            Click on any chunk node in the topology canvas to inspect connected distractor paths.
          </div>
        )}
      </div>
    </div>
  )
}
