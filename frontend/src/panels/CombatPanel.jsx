import { useGameStore } from '../store/gameStore'

const TIER_COLORS = {
  normal: { bg: 'bg-surface-700', text: 'text-surface-300', label: 'Normale' },
  elite:  { bg: 'bg-orange-900',  text: 'text-orange-300',  label: 'Elite' },
  boss:   { bg: 'bg-red-900',     text: 'text-red-300',     label: 'Boss' },
}

const PHASE_COLORS = {
  1: 'text-surface-400',
  2: 'text-orange-400',
  3: 'text-red-400',
}

const PHASE_LABELS = {
  1: 'Fase 1',
  2: 'Fase 2 — Indebolito',
  3: 'Fase 3 — Disperato',
}

function HpBar({ current, max, phase }) {
  const pct = max > 0 ? Math.round((current / max) * 100) : 0
  const barColor =
    pct > 60 ? 'bg-green-600' :
    pct > 30 ? 'bg-yellow-500' :
               'bg-red-600'

  return (
    <div className="flex flex-col gap-1">
      <div className="flex justify-between text-xs font-mono">
        <span className={PHASE_COLORS[phase] ?? 'text-surface-400'}>
          {PHASE_LABELS[phase] ?? 'Fase 1'}
        </span>
        <span className="text-surface-400">{current} / {max}</span>
      </div>
      <div className="w-full h-3 bg-surface-800 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-300 ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

function TensionBar({ value }) {
  const pct = Math.min(100, Math.max(0, value))
  const color =
    pct >= 80 ? 'bg-purple-500' :
    pct >= 50 ? 'bg-yellow-500' :
                'bg-blue-600'
  return (
    <div className="flex flex-col gap-1">
      <div className="flex justify-between text-xs">
        <span className="text-surface-500">Tensione tattica</span>
        <span className={`font-mono ${pct >= 80 ? 'text-purple-400' : 'text-surface-400'}`}>
          {pct}%{pct >= 80 ? ' ⚡ OVERDRIVE' : ''}
        </span>
      </div>
      <div className="w-full h-2 bg-surface-800 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-300 ${color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

function StatusBadge({ effect }) {
  const isDebuff = effect.type === 'debuff'
  return (
    <span
      className={`flex items-center gap-0.5 text-xs px-1.5 py-0.5 rounded border ${
        isDebuff ? 'border-red-800 bg-red-950 text-red-300' : 'border-green-800 bg-green-950 text-green-300'
      }`}
      title={`${effect.name} — ${effect.turns_remaining} turni`}
    >
      {effect.icon} {effect.name}
      <span className="ml-1 opacity-60">{effect.turns_remaining}t</span>
    </span>
  )
}

export default function CombatPanel() {
  const session   = useGameStore((s) => s.session)
  const character = useGameStore((s) => s.character)

  const enemy   = session?.current_enemy
  const tension = session?.tactical_tension ?? 0
  const effects = character?.status_effects ?? []
  const activeEffects = effects.filter((e) => e.turns_remaining > 0)

  if (!enemy) {
    return (
      <div className="h-full flex flex-col items-center justify-center gap-2 text-surface-500">
        <span className="text-2xl">⚔</span>
        <p className="text-xs text-center">Nessun combattimento attivo.<br />Le info sul nemico appariranno qui.</p>
      </div>
    )
  }

  const tier  = TIER_COLORS[enemy.tier] ?? TIER_COLORS.normal
  const phase = enemy.current_phase > 0 ? enemy.current_phase : 1

  return (
    <div className="h-full flex flex-col gap-3 text-sm overflow-y-auto">
      {/* Header nemico */}
      <div className="flex items-start justify-between gap-2 shrink-0">
        <div className="min-w-0">
          <p className="text-surface-100 font-semibold truncate">{enemy.name}</p>
          <p className="text-surface-500 text-xs">Livello {enemy.level}</p>
        </div>
        <span className={`shrink-0 text-xs font-semibold px-2 py-0.5 rounded ${tier.bg} ${tier.text}`}>
          {tier.label}
        </span>
      </div>

      {/* Barra HP con fase */}
      <div className="shrink-0">
        <HpBar current={enemy.hp} max={enemy.max_hp} phase={phase} />
      </div>

      {/* Debolezze e resistenze */}
      {(enemy.weaknesses?.length > 0 || enemy.resistances?.length > 0) && (
        <div className="flex flex-wrap gap-2 shrink-0">
          {enemy.weaknesses?.map((w) => (
            <span key={w} className="text-xs bg-red-950 text-red-300 border border-red-800 px-1.5 py-0.5 rounded">
              ⚠ {w}
            </span>
          ))}
          {enemy.resistances?.map((r) => (
            <span key={r} className="text-xs bg-blue-950 text-blue-400 border border-blue-800 px-1.5 py-0.5 rounded">
              🛡 {r}
            </span>
          ))}
        </div>
      )}

      {/* Tensione tattica */}
      <div className="shrink-0">
        <TensionBar value={tension} />
      </div>

      {/* Status effects del personaggio in combattimento */}
      {activeEffects.length > 0 && (
        <div className="shrink-0">
          <p className="text-xs text-surface-500 mb-1">Effetti attivi su di te</p>
          <div className="flex flex-wrap gap-1">
            {activeEffects.map((e) => (
              <StatusBadge key={e.id} effect={e} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
