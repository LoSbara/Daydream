import { useSkills } from '../hooks/useSkills'
import { useGameStore } from '../store/gameStore'

const RARITY_COLORS = {
  1: 'text-surface-300',
  2: 'text-blue-400',
  3: 'text-purple-400',
}

const TIER_LABELS = { 1: 'Base', 2: 'Avanzata', 3: 'Maestria' }

function SkillCard({ skill, inLoadout, slotsUsed, maxSlots, onToggle, disabled }) {
  const canAdd = !inLoadout && slotsUsed < maxSlots
  const isLocked = !skill.is_unlocked
  const onCooldown = skill.cooldown_remaining > 0

  const borderClass = inLoadout
    ? 'border-accent bg-accent/10'
    : isLocked
    ? 'border-surface-700 opacity-50'
    : 'border-surface-600 hover:border-surface-400'

  return (
    <div className={`rounded border p-2.5 transition-colors ${borderClass}`}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className={`text-xs font-bold ${RARITY_COLORS[skill.tier] ?? 'text-surface-200'}`}>
              {skill.name}
            </span>
            <span className="text-surface-600 text-xs">{TIER_LABELS[skill.tier]}</span>
            {isLocked && (
              <span className="text-surface-600 text-xs">🔒</span>
            )}
            {onCooldown && (
              <span className="text-orange-400 text-xs font-mono">
                CD {skill.cooldown_remaining}t
              </span>
            )}
          </div>
          <p className="text-surface-400 text-xs mt-0.5 leading-tight">{skill.description}</p>
          <div className="flex gap-2 mt-1 text-xs font-mono">
            {skill.mp_cost > 0 && (
              <span className="text-blue-400">MP -{skill.mp_cost}</span>
            )}
            {skill.stm_cost > 0 && (
              <span className="text-green-400">STM -{skill.stm_cost}</span>
            )}
            {skill.cooldown_turns > 0 && (
              <span className="text-surface-500">CD {skill.cooldown_turns}t</span>
            )}
            {skill.damage_multiplier > 0 && (
              <span className="text-red-400">×{skill.damage_multiplier} dmg</span>
            )}
            {skill.heal_amount > 0 && (
              <span className="text-pink-400">+{skill.heal_amount} HP</span>
            )}
          </div>

          {/* Condizione di sblocco */}
          {isLocked && skill.unlock_condition && (
            <p className="text-surface-600 text-xs mt-1 italic">
              {formatUnlockCondition(skill.unlock_condition)}
            </p>
          )}
        </div>

        {/* Bottone loadout */}
        {!isLocked && (
          <button
            onClick={() => onToggle(skill.id)}
            disabled={disabled || (!inLoadout && !canAdd)}
            className={`shrink-0 text-xs px-2 py-1 rounded border transition-colors ${
              inLoadout
                ? 'border-accent text-accent hover:bg-accent/20'
                : canAdd
                ? 'border-surface-500 text-surface-300 hover:border-surface-300'
                : 'border-surface-700 text-surface-600 cursor-not-allowed'
            }`}
          >
            {inLoadout ? '✓ Equipaggiata' : canAdd ? '+ Loadout' : 'Pieno'}
          </button>
        )}
      </div>
    </div>
  )
}

function formatUnlockCondition(cond) {
  const parts = []
  if (cond.enemies_defeated) parts.push(`${cond.enemies_defeated} nemici sconfitti`)
  if (cond.dodges) parts.push(`${cond.dodges} schivate`)
  if (cond.near_death_survives) parts.push(`${cond.near_death_survives} near-death`)
  if (cond.enemies_analyzed) parts.push(`${cond.enemies_analyzed} nemici analizzati`)
  if (cond.criticals) parts.push(`${cond.criticals} critici`)
  return `Sblocca: ${parts.join(', ')}`
}

export default function SkillPanel() {
  const { skills, skillSlots, toggleLoadout } = useSkills()
  const { session, isStreaming } = useGameStore()

  const loadout = session?.skill_loadout ?? []
  const slotsUsed = loadout.length

  if (!skills.length) {
    return (
      <div className="h-full flex items-center justify-center text-surface-500 text-sm">
        Nessuna skill disponibile
      </div>
    )
  }

  // Separa skill in loadout e non
  const inLoadout = skills.filter((s) => s.is_in_loadout)
  const available = skills.filter((s) => !s.is_in_loadout)

  return (
    <div className="h-full overflow-y-auto space-y-3">
      {/* Header slot */}
      <div className="flex items-center justify-between">
        <p className="text-surface-500 text-xs uppercase tracking-wider">Loadout</p>
        <span className="text-xs font-mono text-surface-400">
          {slotsUsed}/{skillSlots} slot
        </span>
      </div>

      {/* Slot bar */}
      <div className="flex gap-1">
        {Array.from({ length: skillSlots }).map((_, i) => (
          <div
            key={i}
            className={`h-1.5 flex-1 rounded ${
              i < slotsUsed ? 'bg-accent' : 'bg-surface-700'
            }`}
          />
        ))}
      </div>

      {/* Skill in loadout */}
      {inLoadout.length > 0 && (
        <div className="space-y-1.5">
          {inLoadout.map((s) => (
            <SkillCard
              key={s.id}
              skill={s}
              inLoadout
              slotsUsed={slotsUsed}
              maxSlots={skillSlots}
              onToggle={toggleLoadout}
              disabled={isStreaming}
            />
          ))}
        </div>
      )}

      {/* Divisore */}
      {available.length > 0 && (
        <>
          <p className="text-surface-600 text-xs uppercase tracking-wider border-t border-surface-700 pt-2">
            Disponibili
          </p>
          <div className="space-y-1.5">
            {available.map((s) => (
              <SkillCard
                key={s.id}
                skill={s}
                inLoadout={false}
                slotsUsed={slotsUsed}
                maxSlots={skillSlots}
                onToggle={toggleLoadout}
                disabled={isStreaming}
              />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
