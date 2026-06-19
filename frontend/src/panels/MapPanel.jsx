import { useGameStore } from '../store/gameStore'

const CELL = 48   // px per stanza
const GAP  = 32   // px tra centri adiacenti
const STEP = CELL + GAP  // distanza tra centri

const DIR_OFFSET = {
  nord:  [0, -1],
  sud:   [0,  1],
  est:   [1,  0],
  ovest: [-1, 0],
}

// BFS dall'ingresso usando le direzioni degli exit per posizionare le stanze in 2D.
// Ritorna: { nodes: [{id, x, y, room}], edges: [{ax,ay,bx,by}] }
function buildGraph(rooms) {
  if (!rooms) return { nodes: [], edges: [] }

  const entrance = Object.values(rooms).find((r) => r.is_entrance)
  if (!entrance) return { nodes: [], edges: [] }

  const pos = {}
  const queue = [{ id: entrance.id, x: 0, y: 0 }]
  const seen = new Set()

  while (queue.length > 0) {
    const { id, x, y } = queue.shift()
    if (seen.has(id)) continue
    seen.add(id)
    pos[id] = { x, y }

    const room = rooms[id]
    for (const [dir, nextId] of Object.entries(room?.exits ?? {})) {
      if (seen.has(nextId)) continue
      const off = DIR_OFFSET[dir]
      if (!off) continue
      const [dx, dy] = off
      const nx = x + dx
      const ny = y + dy
      // Risoluzione conflitti: se la cella è occupata, non sovrascrivere
      // (la stanza verrà aggiunta quando raggiunta da un'altra direzione)
      const occupied = Object.values(pos).some((p) => p.x === nx && p.y === ny)
      if (!occupied) {
        queue.push({ id: nextId, x: nx, y: ny })
      }
    }
  }

  // Stanze non raggiunte dalla BFS (layout disconnesso): aggiungi in coda
  for (const id of Object.keys(rooms)) {
    if (!pos[id]) {
      const maxY = Math.max(...Object.values(pos).map((p) => p.y), -1)
      pos[id] = { x: 0, y: maxY + 1 }
    }
  }

  // Normalizza: trasla in modo che il minimo sia 0
  const xs = Object.values(pos).map((p) => p.x)
  const ys = Object.values(pos).map((p) => p.y)
  const minX = Math.min(...xs)
  const minY = Math.min(...ys)

  const nodes = Object.entries(pos).map(([id, { x, y }]) => ({
    id,
    x: x - minX,
    y: y - minY,
    room: rooms[id],
  }))

  // Archi: un arco per coppia connessa (evita duplicati)
  const edgeSet = new Set()
  const edges = []
  for (const node of nodes) {
    for (const nextId of Object.values(node.room?.exits ?? {})) {
      const key = [node.id, nextId].sort().join('|')
      if (edgeSet.has(key)) continue
      edgeSet.add(key)
      const next = nodes.find((n) => n.id === nextId)
      if (next) {
        edges.push({
          ax: node.x * STEP + CELL / 2,
          ay: node.y * STEP + CELL / 2,
          bx: next.x * STEP + CELL / 2,
          by: next.y * STEP + CELL / 2,
        })
      }
    }
  }

  return { nodes, edges }
}

function roomColor(room, isCurrent) {
  if (isCurrent)                              return { fill: 'var(--accent)',     stroke: 'var(--accent)',     text: '#fff' }
  if (!room?.visited)                         return { fill: '#1a1a1a',           stroke: '#2d2d2d',           text: '#444' }
  if (room.is_boss)                           return { fill: '#3b0000',           stroke: '#7f1d1d',           text: '#fca5a5' }
  if (room.is_entrance)                       return { fill: '#0d2d0d',           stroke: '#166534',           text: '#86efac' }
  if (room.has_enemy && !room.cleared)        return { fill: '#2d1000',           stroke: '#7c2d12',           text: '#fdba74' }
  return                                             { fill: '#1e1e1e',           stroke: '#404040',           text: '#9ca3af' }
}

function roomIcon(room) {
  if (!room?.visited) return '?'
  if (room.is_boss)   return '☠'
  if (room.is_entrance) return '⬛'
  if (room.has_enemy && !room.cleared) return '⚔'
  if (room.cleared)   return '✓'
  return '·'
}

// ── Componente SVG ────────────────────────────────────────────────────────────
function DungeonGraph({ dungeon }) {
  const { nodes, edges } = buildGraph(dungeon?.rooms)
  if (nodes.length === 0) return <p className="text-surface-500 text-xs text-center py-4">Caricamento mappa…</p>

  const maxX = Math.max(...nodes.map((n) => n.x))
  const maxY = Math.max(...nodes.map((n) => n.y))
  const svgW = (maxX + 1) * STEP
  const svgH = (maxY + 1) * STEP

  return (
    <div className="overflow-auto" style={{ maxHeight: '100%' }}>
      <svg
        width={svgW}
        height={svgH}
        viewBox={`0 0 ${svgW} ${svgH}`}
        style={{ display: 'block', minWidth: svgW }}
      >
        {/* Archi */}
        {edges.map((e, i) => (
          <line
            key={i}
            x1={e.ax} y1={e.ay}
            x2={e.bx} y2={e.by}
            stroke="#374151"
            strokeWidth={2}
            strokeDasharray={4}
          />
        ))}

        {/* Nodi */}
        {nodes.map((n) => {
          const isCurrent = n.id === dungeon.current_room
          const cx = n.x * STEP
          const cy = n.y * STEP
          const { fill, stroke, text } = roomColor(n.room, isCurrent)
          const icon = roomIcon(n.room)
          const label = n.room?.visited ? (n.room.name?.split(' ').slice(0, 2).join(' ') ?? '') : '???'

          return (
            <g key={n.id}>
              {/* Alone per stanza corrente */}
              {isCurrent && (
                <rect
                  x={cx - 2} y={cy - 2}
                  width={CELL + 4} height={CELL + 4}
                  rx={6}
                  fill="none"
                  stroke="var(--accent)"
                  strokeWidth={2}
                  opacity={0.4}
                />
              )}
              {/* Box stanza */}
              <rect
                x={cx} y={cy}
                width={CELL} height={CELL}
                rx={4}
                fill={fill}
                stroke={stroke}
                strokeWidth={1.5}
              />
              {/* Icona centrale */}
              <text
                x={cx + CELL / 2} y={cy + CELL / 2 + 1}
                textAnchor="middle"
                dominantBaseline="middle"
                fontSize={14}
                fill={text}
                fontFamily="monospace"
              >
                {icon}
              </text>
              {/* Label sotto */}
              <text
                x={cx + CELL / 2} y={cy + CELL + 11}
                textAnchor="middle"
                dominantBaseline="middle"
                fontSize={8}
                fill="#6b7280"
                fontFamily="monospace"
              >
                {label.slice(0, 12)}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

// ── Legenda ───────────────────────────────────────────────────────────────────
function Legend() {
  const items = [
    { color: 'var(--accent)', label: 'Posizione' },
    { color: '#166534',       label: 'Ingresso' },
    { color: '#7f1d1d',       label: 'Boss' },
    { color: '#7c2d12',       label: 'Nemico' },
    { color: '#404040',       label: 'Esplorata' },
    { color: '#2d2d2d',       label: 'Inesplorata' },
  ]
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1 pt-2 border-t border-surface-800">
      {items.map(({ color, label }) => (
        <span key={label} className="flex items-center gap-1 text-xs text-surface-500">
          <span className="w-2.5 h-2.5 rounded-sm shrink-0" style={{ background: color }} />
          {label}
        </span>
      ))}
    </div>
  )
}

// ── Panel principale ──────────────────────────────────────────────────────────
export default function MapPanel() {
  const session = useGameStore((s) => s.session)
  const dungeon = session?.active_dungeon

  if (!dungeon) {
    return (
      <div className="h-full flex flex-col items-center justify-center gap-2 text-surface-500">
        <span className="text-2xl">🗺</span>
        <p className="text-xs text-center">Nessun dungeon attivo.<br />Entra in un dungeon per vedere la mappa.</p>
      </div>
    )
  }

  const rooms = dungeon.rooms ?? {}
  const totalRooms = Object.keys(rooms).length
  const visitedRooms = Object.values(rooms).filter((r) => r.visited).length

  return (
    <div className="h-full flex flex-col gap-2 text-sm">
      {/* Header */}
      <div className="flex items-center justify-between shrink-0">
        <p className="text-surface-100 text-xs font-semibold truncate">{dungeon.name}</p>
        <span className="text-surface-500 text-xs font-mono shrink-0 ml-2">
          {visitedRooms}/{totalRooms}
        </span>
      </div>

      {/* Mappa SVG */}
      <div className="flex-1 min-h-0 overflow-auto">
        <DungeonGraph dungeon={dungeon} />
      </div>

      {/* Legenda */}
      <Legend />
    </div>
  )
}
