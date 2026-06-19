import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

const DISPOSITION_STYLE = {
  amichevole: { dot: 'bg-green-500', label: 'Amichevole' },
  neutrale:   { dot: 'bg-yellow-500', label: 'Neutrale' },
  ostile:     { dot: 'bg-red-500',  label: 'Ostile' },
}

function NPCCard({ npc }) {
  const disp = DISPOSITION_STYLE[npc.disposition] ?? DISPOSITION_STYLE.neutrale
  return (
    <div className="border border-surface-700 rounded p-2 flex flex-col gap-1 hover:border-surface-500 transition-colors">
      <div className="flex items-center justify-between gap-2">
        <p className="text-surface-100 text-xs font-semibold truncate">{npc.name}</p>
        <span className={`shrink-0 w-2 h-2 rounded-full ${disp.dot}`} title={disp.label} />
      </div>
      {npc.role && (
        <p className="text-surface-500 text-xs italic truncate">{npc.role}</p>
      )}
      {npc.description && (
        <p className="text-surface-400 text-xs leading-snug line-clamp-2">{npc.description}</p>
      )}
      {npc.location && (
        <p className="text-surface-600 text-xs">📍 {npc.location}</p>
      )}
    </div>
  )
}

export default function NPCPanel() {
  const session  = useGameStore((s) => s.session)
  const [npcs, setNpcs]       = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState(null)

  function fetchNPCs() {
    setLoading(true)
    setError(null)
    api.get('/catalog/npcs')
      .then((res) => setNpcs(res.npcs ?? []))
      .catch(() => setError('Impossibile caricare gli NPC.'))
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchNPCs() }, [session?.location])

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center text-surface-500 text-xs">
        Caricamento…
      </div>
    )
  }

  if (error) {
    return (
      <div className="h-full flex flex-col items-center justify-center gap-2 text-surface-500">
        <p className="text-xs">{error}</p>
        <button onClick={fetchNPCs} className="text-xs text-accent hover:underline">Riprova</button>
      </div>
    )
  }

  if (npcs.length === 0) {
    return (
      <div className="h-full flex flex-col items-center justify-center gap-2 text-surface-500">
        <span className="text-2xl">👤</span>
        <p className="text-xs text-center">Nessun NPC conosciuto.<br />Interagisci con i personaggi del mondo.</p>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col gap-2 text-sm">
      <div className="flex items-center justify-between shrink-0">
        <p className="text-surface-400 text-xs">{npcs.length} personagg{npcs.length === 1 ? 'io' : 'i'} conosciut{npcs.length === 1 ? 'o' : 'i'}</p>
        <button onClick={fetchNPCs} className="text-xs text-surface-500 hover:text-accent transition-colors">↻</button>
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto flex flex-col gap-2">
        {npcs.map((npc) => (
          <NPCCard key={npc.id ?? npc.name} npc={npc} />
        ))}
      </div>
    </div>
  )
}
