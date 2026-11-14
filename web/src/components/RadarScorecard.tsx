import React from 'react'
import type { AttackVector } from '../types'

interface RadarScorecardProps {
  scores: Record<AttackVector, number>
}

const vectorMeta: { id: AttackVector; label: string }[] = [
  { id: 'negation_inversion', label: 'Negation Inversion' },
  { id: 'semantic_distractor', label: 'Semantic Distractor' },
  { id: 'entity_version_collision', label: 'Entity / Version' },
  { id: 'orthographic_noise', label: 'Typo Noise' },
  { id: 'contradiction_trap', label: 'Contradiction Trap' },
  { id: 'out_of_domain', label: 'Out of Domain' },
]

export const RadarScorecard: React.FC<RadarScorecardProps> = ({ scores }) => {
  const cx = 170
  const cy = 160
  const r = 110
  const totalAxes = vectorMeta.length

  // Calculate polygon points
  const points: { x: number; y: number; label: string; score: number; vecId: AttackVector }[] = []

  vectorMeta.forEach((v, i) => {
    const angle = i * ((2 * Math.PI) / totalAxes) - Math.PI / 2
    const score = Math.max(0, Math.min(100, scores[v.id] ?? 0))
    const ratio = score / 100
    const x = cx + r * ratio * Math.cos(angle)
    const y = cy + r * ratio * Math.sin(angle)
    points.push({ x, y, label: v.label, score, vecId: v.id })
  })

  const polygonCoords = points.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')

  return (
    <div className="flex flex-col items-center">
      <div className="relative">
        <svg width="340" height="320" viewBox="0 0 340 320" className="overflow-visible">
          {/* Concentric grid rings */}
          {[0.25, 0.5, 0.75, 1.0].map((pct, idx) => {
            const ringPts = vectorMeta
              .map((_, i) => {
                const angle = i * ((2 * Math.PI) / totalAxes) - Math.PI / 2
                const px = cx + r * pct * Math.cos(angle)
                const py = cy + r * pct * Math.sin(angle)
                return `${px.toFixed(1)},${py.toFixed(1)}`
              })
              .join(' ')
            return (
              <polygon
                key={idx}
                points={ringPts}
                fill="none"
                stroke="#1e293b"
                strokeWidth={idx === 3 ? '1.5' : '1'}
              />
            )
          })}

          {/* Spoke lines and labels */}
          {vectorMeta.map((v, i) => {
            const angle = i * ((2 * Math.PI) / totalAxes) - Math.PI / 2
            const ax = cx + r * Math.cos(angle)
            const ay = cy + r * Math.sin(angle)
            const lx = cx + (r + 28) * Math.cos(angle)
            const ly = cy + (r + 28) * Math.sin(angle)

            let textAnchor: 'middle' | 'start' | 'end' = 'middle'
            if (Math.abs(Math.cos(angle)) > 0.3) {
              textAnchor = Math.cos(angle) > 0 ? 'start' : 'end'
            }

            return (
              <g key={v.id}>
                <line
                  x1={cx}
                  y1={cy}
                  x2={ax}
                  y2={ay}
                  stroke="#334155"
                  strokeWidth="1"
                  strokeDasharray="3,3"
                />
                <text
                  x={lx}
                  y={ly}
                  fill="#94a3b8"
                  fontSize="11"
                  textAnchor={textAnchor}
                  alignmentBaseline="middle"
                  className="font-medium select-none"
                >
                  {v.label}
                </text>
              </g>
            )
          })}

          {/* Shaded Area */}
          <polygon
            points={polygonCoords}
            fill="rgba(56, 189, 248, 0.25)"
            stroke="#38bdf8"
            strokeWidth="2.5"
            className="transition-all duration-300"
          />

          {/* Interactive vertex dots */}
          {points.map((p) => (
            <circle
              key={p.vecId}
              cx={p.x}
              cy={p.y}
              r="4.5"
              fill={p.score >= 80 ? '#10b981' : p.score >= 60 ? '#f59e0b' : '#ef4444'}
              stroke="#0b0f19"
              strokeWidth="2"
            />
          ))}
        </svg>
      </div>

      <div className="grid grid-cols-2 gap-2 mt-4 w-full">
        {points.map((p) => {
          const isHigh = p.score >= 85
          const isMed = p.score >= 70 && p.score < 85
          const colorClass = isHigh ? 'text-emerald-400' : isMed ? 'text-amber-400' : 'text-rose-400'
          return (
            <div
              key={p.vecId}
              className="flex justify-between items-center text-xs bg-slate-900/60 border border-slate-800/80 px-2.5 py-1.5 rounded"
            >
              <span className="text-slate-400 truncate mr-2">{p.label}</span>
              <span className={`font-semibold ${colorClass}`}>{p.score.toFixed(1)}%</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
