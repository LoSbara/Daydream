import { useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

// Costruisce l'ordine lineare delle stanze tramite BFS dall'ingresso.
// Restituisce un array di room objects in ordine di scoperta.
function buildRoomChain(rooms) {
  if (!rooms) return []
  const entrance = Object.values(rooms).find((r) => r.is_entrance)
  if (!entrance) return Object.values(rooms)

  const ordered = []
  const seen = new Set()
  const queue = [entrance.id]

  while (queue.length > 0) {
    const id = queue.shift()
    if (seen.has(id)) continue
    seen.add(id)
    ordered.push(rooms[id])
    for (const nextId of Object.values(rooms[id]?.exits ?? {})) {
      if (!seen.has(nextId)) queue.push(nextId)
    }
  }
  return ordered
}

const DIFFICULTIES = [1, 2, 3, 4, 5]

const DUNGEON_LABELS = {
  caverna_oscura:    'Caverna Oscura',
  torre_abbandonata: 'Torre Abbandonata',
  rovine_antiche:    'Rovine Antiche',
}

const TIER_COLORS = {
  normal: 'text-red-400',
  elite:  'text-orange-400',
  boss:   'text-red-300',
}

// ── Nessun dungeon attivo: selezione e ingresso ───────────────────────────────
function EnterDungeonView() {
  const { session, updateSession } = useGameStore()
  const [selected, setSelected] = useState('caverna_oscura')
  const [difficulty, setDifficulty] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const inCombat = session?.combat_active

  async function handleEnter() {
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/dungeon/enter', { dungeon_id: selected, difficulty })
      updateSession(res.session)
    } catch (err) {
      setError(err.message || 'Errore ingresso dungeon')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <p className="text-surface-400 text-xs">
        Scegli un dungeon e una difficoltà. La navigazione avviene tramite chat.
      </p>

      {/* Selezione dungeon */}
      <div className="space-y-1.5">
        {Object.entries(DUNGEON_LABELS).map(([id, label]) => (
          <button
            key={id}
            onClick={() => setSelected(id)}
            className={`w-full text-left px-3 py-2 rounded border text-sm transition-colors ${
              selected === id
                ? 'border-accent bg-accent/10 text-surface-100'
                : 'border-surface-700 text-surface-400 hover:border-surface-500'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Difficoltà */}
      <div>
        <p className="text-surface-500 text-xs mb-1.5">Difficoltà</p>
        <div className="flex gap-1.5">
          {DIFFICULTIES.map((d) => (
            <button
              key={d}
              onClick={() => setDifficulty(d)}
              className={`w-8 h-8 rounded border text-sm font-mono transition-colors ${
                difficulty === d
                  ? 'border-accent bg-accent/20 text-accent'
                  : 'border-surface-700 text-surface-500 hover:border-surface-500'
              }`}
            >
              {d}
            </button>
          ))}
        </div>
        <p className="text-surface-600 text-xs mt-1">
          {difficulty <= 2 ? 'Facile — buono per iniziare' :
           difficulty <= 3 ? 'Medio — sfida equilibrata' :
           difficulty <= 4 ? 'Difficile — alta densità nemici' :
                             'Estremo — solo per chi è pronto'}
        </p>
      </div>

      {error && <p className="text-red-400 text-xs">{error}</p>}

      {inCombat && (
        <p className="text-yellow-400 text-xs">Non puoi entrare durante un combattimento.</p>
      )}

      <button
        onClick={handleEnter}
        disabled={loading || inCombat}
        className="w-full py-2 rounded border border-accent text-accent text-sm hover:bg-accent/10 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? 'Entrata in corso…' : `Entra in ${DUNGEON_LABELS[selected]}`}
      </button>
    </div>
  )
}

// ── Dungeon attivo: mappa e stanza corrente ───────────────────────────────────
function ActiveDungeonView({ dungeon }) {
  const { session, updateSession } = useGameStore()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const chain = buildRoomChain(dungeon.rooms)
  const currentRoom = dungeon.rooms[dungeon.current_room]
  const atEntrance = currentRoom?.is_entrance

  async function handleExit() {
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/dungeon/exit', {})
      updateSession(res.session)
    } catch (err) {
      setError(err.message || 'Errore uscita dungeon')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-3">
      {/* Header dungeon */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-surface-100 text-sm font-semibold">{dungeon.name}</p>
          <div className="flex gap-0.5 mt-0.5">
            {DIFFICULTIES.map((d) => (
              <span
                key={d}
                className={`text-xs ${d <= dungeon.difficulty ? 'text-accent' : 'text-surface-700'}`}
              >
                ◆
              </span>
            ))}
          </div>
        </div>
        <span className="text-surface-500 text-xs font-mono">
          {chain.filter((r) => r.visited).length}/{chain.length} stanze
        </span>
      </div>

      {/* Mappa lineare */}
      <div className="overflow-x-auto pb-1">
        <div className="flex items-center gap-0 min-w-max">
          {chain.map((room, idx) => {
            const isCurrent = room.id === dungeon.current_room
            const isVisited = room.visited

            let nodeColor = 'bg-surface-700 border-surface-600 text-surface-500'
            if (isCurrent)       nodeColor = 'bg-accent border-accent text-white'
            else if (room.is_entrance && isVisited) nodeColor = 'bg-green-900 border-green-600 text-green-300'
            else if (room.is_boss && isVisited) nodeColor = 'bg-red-900 border-red-600 text-red-300'
            else if (isVisited)  nodeColor = 'bg-surface-800 border-surface-500 text-surface-300'

            const icon = room.is_boss ? '☠' :
                         room.is_entrance ? '⬛' :
                         (room.has_enemy && !room.cleared) ? '⚔' : '·'

            return (
              <div key={room.id} className="flex items-center">
                <div
                  title={isVisited ? room.name : '???'}
                  className={`w-7 h-7 rounded border flex items-center justify-center text-xs select-none
                    ${nodeColor} ${isCurrent ? 'shadow-[0_0_6px_rgba(var(--accent),0.6)]' : ''}`}
                >
                  {isVisited ? icon : '?'}
                </div>
                {idx < chain.length - 1 && (
                  <div className={`w-3 h-px ${
                    chain[idx + 1]?.visited ? 'bg-surface-500' : 'bg-surface-700'
                  }`} />
                )}
              </div>
            )
          })}
        </div>
      </div>

      {/* Stanza corrente */}
      {currentRoom && (
        <div className="border border-surface-700 rounded p-2.5 space-y-2">
          <div className="flex items-start justify-between gap-2">
            <p className="text-surface-100 text-xs font-semibold leading-tight">
              {currentRoom.is_boss && <span className="text-red-400 mr-1">☠</span>}
              {currentRoom.is_entrance && <span className="text-green-400 mr-1">⬛</span>}
              {currentRoom.name}
            </p>
            {currentRoom.has_enemy && !currentRoom.cleared && (
              <span className={`text-xs shrink-0 ${TIER_COLORS[currentRoom.enemy_tier] ?? 'text-red-400'}`}>
                ⚔ {currentRoom.enemy_tier}
              </span>
            )}
            {currentRoom.cleared && (
              <span className="text-green-600 text-xs shrink-0">✓ libera</span>
            )}
          </div>

          <p className="text-surface-400 text-xs leading-relaxed line-clamp-2">
            {currentRoom.description}
          </p>

          {/* Uscite */}
          {Object.keys(currentRoom.exits ?? {}).length > 0 && (
            <div className="flex flex-wrap gap-1">
              {Object.entries(currentRoom.exits).map(([dir, roomId]) => {
                const target = dungeon.rooms[roomId]
                const hasEnemy = target?.has_enemy && !target?.cleared
                return (
                  <span
                    key={dir}
                    className={`px-1.5 py-0.5 rounded border text-xs font-mono ${
                      hasEnemy
                        ? 'border-red-800 text-red-400 bg-red-950/20'
                        : 'border-surface-600 text-surface-400'
                    }`}
                  >
                    {dir}{hasEnemy ? ' ⚠' : ''}
                  </span>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Hint navigazione */}
      <p className="text-surface-600 text-xs italic">
        Scrivi nel chat per muoverti — es. "vado a nord"
      </p>

      {error && <p className="text-red-400 text-xs">{error}</p>}

      {/* Pulsante uscita */}
      {atEntrance && (
        <button
          onClick={handleExit}
          disabled={loading || session?.combat_active}
          className="w-full py-1.5 rounded border border-surface-600 text-surface-400 text-xs hover:border-surface-400 hover:text-surface-200 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          {loading ? 'Uscita…' : 'Esci dal dungeon'}
        </button>
      )}
      {!atEntrance && (
        <p className="text-surface-600 text-xs text-center">
          Torna all'ingresso per uscire
        </p>
      )}
    </div>
  )
}

// ── Panel principale ──────────────────────────────────────────────────────────
export default function DungeonPanel() {
  const session = useGameStore((s) => s.session)
  const dungeon = session?.active_dungeon

  return (
    <div className="text-sm h-full overflow-auto">
      {dungeon
        ? <ActiveDungeonView dungeon={dungeon} />
        : <EnterDungeonView />
      }
    </div>
  )
}
