import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

const SCOPE_META = {
  world:   { label: 'Mondo',       color: 'text-purple-400' },
  kingdom: { label: 'Regno',       color: 'text-blue-400' },
  city:    { label: 'Città',       color: 'text-green-400' },
  npc:     { label: 'NPC',         color: 'text-yellow-400' },
  faction: { label: 'Fazione',     color: 'text-orange-400' },
  dungeon: { label: 'Dungeon',     color: 'text-red-400' },
  player:  { label: 'Personaggio', color: 'text-cyan-400' },
}

function getScopeType(scope) {
  const i = scope.indexOf(':')
  return i === -1 ? scope : scope.slice(0, i)
}
function getScopeName(scope) {
  const i = scope.indexOf(':')
  return i === -1 ? '' : scope.slice(i + 1).replace(/_/g, ' ')
}

export default function WorldPanel() {
  const turnId = useGameStore((s) => s.session?.turn_id)
  const [flags, setFlags]     = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api.get('/world/flags')
      .then((data) => setFlags(Array.isArray(data) ? data : []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [turnId])

  if (loading) return <p className="text-surface-500 text-xs p-2">Caricamento…</p>

  if (flags.length === 0) return (
    <div className="flex flex-col items-center justify-center h-full text-center p-4">
      <p className="text-3xl mb-2">🌍</p>
      <p className="text-surface-500 text-xs">Nessun evento registrato.</p>
      <p className="text-surface-700 text-xs mt-1">Il GM creerà flag quando accadranno eventi significativi.</p>
    </div>
  )

  const grouped = {}
  for (const f of flags) {
    if (!grouped[f.scope]) grouped[f.scope] = []
    grouped[f.scope].push(f)
  }

  return (
    <div className="space-y-4 text-xs">
      {Object.entries(grouped).map(([scope, items]) => {
        const type = getScopeType(scope)
        const name = getScopeName(scope)
        const meta = SCOPE_META[type] ?? { label: type.toUpperCase(), color: 'text-surface-400' }
        return (
          <div key={scope}>
            <div className="flex items-center gap-1.5 mb-1.5">
              <span className={`font-bold uppercase tracking-wider text-xs ${meta.color}`}>{meta.label}</span>
              {name && <span className="text-surface-400 capitalize">{name}</span>}
            </div>
            <div className="space-y-1.5 pl-2 border-l-2 border-surface-700">
              {items.map((f) => (
                <div key={f.key}>
                  <div className="flex justify-between gap-2">
                    <span className="text-surface-400">{f.key.replace(/_/g, ' ')}</span>
                    <span className="text-surface-100 font-mono">{f.value}</span>
                  </div>
                  {f.description && <p className="text-surface-600 leading-tight mt-0.5">{f.description}</p>}
                </div>
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
