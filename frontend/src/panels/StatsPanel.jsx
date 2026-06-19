import { useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

function ResourceBar({ label, current, max, color }) {
  const pct = max > 0 ? Math.round((current / max) * 100) : 0
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs font-mono">
        <span className="text-surface-300">{label}</span>
        <span style={{ color }}>
          {current}/{max}
        </span>
      </div>
      <div className="h-2 rounded bg-surface-700 overflow-hidden">
        <div
          className="h-full rounded transition-all duration-500"
          style={{ width: `${pct}%`, background: color }}
        />
      </div>
    </div>
  )
}

function TacticalTensionBar({ tension, overdrive }) {
  const pct = Math.min(tension, 100)
  const color = pct >= 80 ? '#f97316' : pct >= 50 ? '#eab308' : '#60a5fa'
  return (
    <div className="space-y-0.5">
      <div className="flex justify-between text-xs font-mono">
        <span className={`${overdrive ? 'text-orange-400 animate-pulse font-bold' : 'text-surface-300'}`}>
          {overdrive ? '⚡ OVERDRIVE' : 'Tension'}
        </span>
        <span style={{ color }}>{tension}/100</span>
      </div>
      <div className="h-1.5 rounded bg-surface-700 overflow-hidden">
        <div
          className={`h-full rounded transition-all duration-500 ${overdrive ? 'animate-pulse' : ''}`}
          style={{ width: `${pct}%`, background: color }}
        />
      </div>
    </div>
  )
}

// Riga stat con pulsante + opzionale per l'allocazione
function StatRow({ label, value, hint, canAllocate, onAllocate, loading }) {
  return (
    <div className="flex justify-between items-center text-xs">
      <div>
        <span className="text-surface-400">{label}</span>
        {hint && <span className="text-surface-600 ml-1">{hint}</span>}
      </div>
      <div className="flex items-center gap-1.5">
        <span className="text-surface-100 font-mono font-bold">{value}</span>
        {canAllocate && (
          <button
            onClick={() => onAllocate(label)}
            disabled={loading}
            className="w-5 h-5 rounded border border-accent/60 text-accent text-xs
                       hover:bg-accent/20 hover:border-accent disabled:opacity-30
                       flex items-center justify-center transition-colors"
          >
            +
          </button>
        )}
      </div>
    </div>
  )
}

export default function StatsPanel() {
  const { character, session, levelUpInfo, clearLevelUpInfo, overdrive, updateCharacter } = useGameStore()
  const [allocating, setAllocating] = useState(false)
  const [allocError, setAllocError] = useState(null)

  if (!character) {
    return (
      <div className="flex items-center justify-center text-surface-500 text-sm py-8">
        Nessun personaggio caricato
      </div>
    )
  }

  const { stats, name, job, level, experience, experience_to_next, money, stat_points_available } = character
  const hasPoints = (stat_points_available ?? 0) > 0

  async function handleAllocate(attribute) {
    setAllocating(true)
    setAllocError(null)
    try {
      const res = await api.put('/character/stats', { attribute, points: 1 })
      updateCharacter(res)
    } catch (err) {
      setAllocError(err.message || 'Errore allocazione')
    } finally {
      setAllocating(false)
    }
  }

  // Hint visibile per le stat che influenzano i max
  const statHints = { VIT: '(+5 HP max)', TEC: '(+3 MP max)' }

  return (
    <div className="space-y-4 text-sm">
      {/* Level-up banner */}
      {levelUpInfo && (
        <div
          className="bg-accent/20 border border-accent rounded p-2 text-center text-accent text-xs cursor-pointer animate-pulse"
          onClick={clearLevelUpInfo}
        >
          ⬆ LEVEL UP! Sei ora livello {levelUpInfo.newLevel} — clicca per chiudere
        </div>
      )}

      {/* Badge punti stat disponibili */}
      {hasPoints && (
        <div className="bg-amber-900/30 border border-amber-600/50 rounded p-2 text-center">
          <p className="text-amber-400 text-xs font-semibold">
            {stat_points_available} punt{stat_points_available === 1 ? 'o' : 'i'} stat da distribuire
          </p>
          <p className="text-amber-600 text-xs mt-0.5">Premi + accanto a una statistica</p>
        </div>
      )}

      {/* Intestazione */}
      <div>
        <p className="text-surface-100 font-bold text-base">{name}</p>
        <p className="text-surface-400 text-xs">
          {job}{character.subclass ? ` / ${character.subclass}` : ''} — Lv.{level}
        </p>
        <div className="mt-1">
          <div className="h-1 rounded bg-surface-700 overflow-hidden">
            <div
              className="h-full rounded bg-accent transition-all duration-700"
              style={{ width: `${experience_to_next > 0 ? Math.round((experience / experience_to_next) * 100) : 0}%` }}
            />
          </div>
          <p className="text-surface-500 text-xs mt-0.5">{experience}/{experience_to_next} EXP</p>
        </div>
      </div>

      {/* Risorse */}
      <div className="space-y-2">
        <ResourceBar label="HP" current={stats.HP.current} max={stats.HP.max} color="hsl(var(--hp))" />
        <ResourceBar label="MP" current={stats.MP.current} max={stats.MP.max} color="hsl(var(--mp))" />
        <ResourceBar label="STM" current={stats.STM.current} max={stats.STM.max} color="hsl(var(--stm))" />
      </div>

      {/* Stats base */}
      <div className="border-t border-surface-700 pt-3 space-y-1.5">
        <p className="text-surface-500 text-xs uppercase tracking-wider mb-2">Statistiche</p>
        {['STR', 'DEX', 'AGI', 'TEC', 'VIT', 'LUC'].map((attr) => (
          <StatRow
            key={attr}
            label={attr}
            value={stats[attr]}
            hint={statHints[attr]}
            canAllocate={hasPoints}
            onAllocate={handleAllocate}
            loading={allocating}
          />
        ))}
        {allocError && <p className="text-red-400 text-xs">{allocError}</p>}
      </div>

      {/* Gold */}
      <div className="border-t border-surface-700 pt-3">
        <div className="flex justify-between text-xs">
          <span className="text-surface-400">Gold</span>
          <span className="text-yellow-400 font-mono font-bold">{money.toLocaleString()}</span>
        </div>
      </div>

      {/* Posizione */}
      {session && (
        <div className="border-t border-surface-700 pt-3">
          <p className="text-surface-500 text-xs uppercase tracking-wider mb-1">Posizione</p>
          <p className="text-surface-200 text-xs">{session.location}</p>
          {session.sub_location && (
            <p className="text-surface-400 text-xs">{session.sub_location}</p>
          )}
          <p className="text-xs mt-1">
            <span className={`px-1.5 py-0.5 rounded text-xs font-mono ${
              session.zone_type === 'safe_zone' ? 'bg-green-900/40 text-green-400' :
              session.zone_type === 'dungeon' ? 'bg-purple-900/40 text-purple-400' :
              'bg-red-900/40 text-red-400'
            }`}>
              {session.zone_type}
            </span>
          </p>
        </div>
      )}

      {/* Tactical Tension (solo in combattimento) */}
      {session?.combat_active && (
        <div className="border-t border-surface-700 pt-3">
          <TacticalTensionBar tension={session.tactical_tension ?? 0} overdrive={overdrive} />
        </div>
      )}

      {/* Combattimento */}
      {session?.combat_active && session.current_enemy && (
        <div className="border border-red-800 rounded p-2 bg-red-950/30">
          <p className="text-red-400 text-xs font-bold uppercase tracking-wider mb-1">⚔ In combattimento</p>
          <p className="text-surface-100 text-xs">{session.current_enemy.name}</p>
          <div className="mt-1">
            <div className="h-1.5 rounded bg-surface-700 overflow-hidden">
              <div
                className="h-full rounded bg-red-500 transition-all duration-500"
                style={{
                  width: `${session.current_enemy.max_hp > 0
                    ? Math.round((session.current_enemy.hp / session.current_enemy.max_hp) * 100)
                    : 0}%`
                }}
              />
            </div>
            <p className="text-surface-400 text-xs mt-0.5">
              {session.current_enemy.hp}/{session.current_enemy.max_hp} HP
            </p>
          </div>
        </div>
      )}

      {/* Status effects */}
      {character.status_effects?.length > 0 && (
        <div className="border-t border-surface-700 pt-3">
          <p className="text-surface-500 text-xs uppercase tracking-wider mb-2">Status</p>
          <div className="flex flex-wrap gap-1">
            {character.status_effects.map((se) => (
              <span
                key={se.id}
                className={`px-1.5 py-0.5 rounded text-xs ${
                  se.type === 'buff' ? 'bg-green-900/40 text-green-400' : 'bg-red-900/40 text-red-400'
                }`}
              >
                {se.name} ({se.turns_remaining})
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
