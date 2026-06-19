import { useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

const RARITY_COLORS = {
  common:    'text-surface-400',
  uncommon:  'text-green-400',
  rare:      'text-blue-400',
  epic:      'text-purple-400',
  legendary: 'text-yellow-400',
}
const RARITY_BORDER = {
  common:    'border-surface-600',
  uncommon:  'border-green-700',
  rare:      'border-blue-700',
  epic:      'border-purple-700',
  legendary: 'border-yellow-600',
}
const RARITY_LABELS = {
  common: 'Comune', uncommon: 'Non comune', rare: 'Raro',
  epic: 'Epico', legendary: 'Leggendario',
}
const TYPE_ICONS = {
  weapon: '⚔', armor: '🛡', offhand: '🔮', accessory: '💍',
  head: '⛑', legs: '🩱', boots: '👢', consumable: '🧪', material: '💎',
}
const SLOT_LABELS = {
  weapon: 'Arma', offhand: 'Mano sinistra', head: 'Testa',
  chest: 'Petto', legs: 'Gambe', boots: 'Stivali',
  accessory_1: 'Acc. 1', accessory_2: 'Acc. 2',
}
const STAT_LABELS = {
  STR: 'FOR', DEX: 'DES', AGI: 'AGI', TEC: 'TEC', VIT: 'VIT', LUC: 'LUC',
}

function StatBadges({ statBonus, perceivedStatBonus, appraised, rarity }) {
  const isUnknown = appraised === false && rarity && rarity !== 'common'
  if (isUnknown) {
    return (
      <div className="flex flex-wrap gap-1 mt-1">
        <span className="text-xs bg-surface-700 px-1.5 py-0.5 rounded font-mono text-surface-600 select-none" style={{ filter: 'blur(3px)' }}>+? ???</span>
        <span className="text-xs bg-surface-700 px-1.5 py-0.5 rounded font-mono text-surface-600 select-none" style={{ filter: 'blur(3px)' }}>+? ???</span>
      </div>
    )
  }
  // Se appraised e ha stat percepite (analisi imperfetta), mostra perceived con "~"
  const displayStats = appraised && perceivedStatBonus && Object.keys(perceivedStatBonus).length > 0
    ? perceivedStatBonus
    : statBonus
  const isEstimate = appraised && perceivedStatBonus && Object.keys(perceivedStatBonus).length > 0
  if (!displayStats || Object.keys(displayStats).length === 0) return null
  return (
    <div className="flex flex-wrap gap-1 mt-1">
      {Object.entries(displayStats).map(([k, v]) => (
        <span key={k} className={`text-xs bg-surface-700 px-1.5 py-0.5 rounded font-mono ${isEstimate ? 'text-yellow-400/70' : 'text-surface-300'}`}>
          {isEstimate && '~'}{v > 0 ? '+' : ''}{v} {STAT_LABELS[k] ?? k}
        </span>
      ))}
    </div>
  )
}

function ItemCard({ item, onEquip, onUnequip, onAppraise, equipped = false, loading }) {
  const color = RARITY_COLORS[item.rarity] ?? 'text-surface-400'
  const border = RARITY_BORDER[item.rarity] ?? 'border-surface-600'
  const icon = TYPE_ICONS[item.type] ?? '📦'
  const isUnidentified = item.appraised === false && item.rarity && item.rarity !== 'common'

  return (
    <div className={`rounded border p-2 ${border} bg-surface-900/50`}>
      <div className="flex items-start gap-1.5">
        <span className="text-base leading-none mt-0.5">{icon}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className={`text-xs font-semibold ${color} leading-tight`}>{item.name}</span>
            <span className={`text-xs ${color} opacity-60`}>({RARITY_LABELS[item.rarity] ?? item.rarity})</span>
          </div>
          <StatBadges statBonus={item.stat_bonus} perceivedStatBonus={item.perceived_stat_bonus} appraised={item.appraised} rarity={item.rarity} />
          {isUnidentified && !equipped && (
            <button
              onClick={(e) => { e.stopPropagation(); onAppraise && onAppraise(item.id) }}
              disabled={!!loading}
              className="text-xs px-1.5 py-0.5 rounded border border-yellow-700 text-yellow-500 hover:bg-yellow-900/20 disabled:opacity-40 transition-colors mt-1 w-full"
            >
              🔍 Identifica
            </button>
          )}
        </div>
        {item.slot && !equipped && (
          <button
            onClick={() => onEquip(item)}
            disabled={!!loading}
            className="text-xs px-1.5 py-0.5 rounded border border-accent text-accent hover:bg-accent/10 disabled:opacity-40 shrink-0"
          >
            Equip
          </button>
        )}
        {equipped && (
          <button
            onClick={() => onUnequip && onUnequip(item.slot)}
            disabled={!!loading}
            className="text-xs px-1.5 py-0.5 rounded border border-surface-600 text-surface-500 hover:border-red-700 hover:text-red-400 disabled:opacity-40 transition-colors ml-auto shrink-0"
          >
            Rimuovi
          </button>
        )}
      </div>
    </div>
  )
}

export default function InventoryPanel() {
  const inventory = useGameStore((s) => s.inventory)
  const setInventory = useGameStore((s) => s.setInventory)
  const updateCharacter = useGameStore((s) => s.updateCharacter)
  const [loading, setLoading] = useState(null)

  if (!inventory) return <p className="text-surface-500 text-xs p-2">Inventario non disponibile.</p>

  const equipped = inventory.equipped ?? {}
  const bag = inventory.bag ?? []

  const equippedSlots = Object.entries(SLOT_LABELS)
    .map(([slot, label]) => ({ slot, label, item: equipped[slot] }))
    .filter(({ item }) => !!item)

  async function handleEquip(item) {
    if (!item.slot) return
    setLoading(item.id)
    try {
      const res = await api.post('/inventory/equip', { item_id: item.id, slot: item.slot })
      setInventory(res.inventory ?? res)
    } catch {}
    setLoading(null)
  }

  async function handleUnequip(slot) {
    setLoading(slot)
    try {
      const res = await api.post('/inventory/unequip', { slot })
      setInventory(res.inventory ?? res)
    } catch {}
    setLoading(null)
  }

  async function handleAppraise(itemId) {
    setLoading(itemId)
    try {
      const res = await api.post('/inventory/appraise', { item_id: itemId })
      setInventory(res.inventory ?? res)
      if (res.character) updateCharacter(res.character)
    } catch {}
    setLoading(null)
  }

  return (
    <div className="space-y-3 text-xs">
      {/* Equipaggiato */}
      {equippedSlots.length > 0 && (
        <div>
          <p className="text-surface-500 uppercase tracking-wider mb-1.5">Equipaggiato</p>
          <div className="space-y-1.5">
            {equippedSlots.map(({ slot, label, item }) => (
              <div key={slot} className="flex items-start gap-1.5">
                <span className="text-surface-600 w-16 shrink-0">{label}</span>
                <div className="flex-1">
                  <ItemCard item={item} onEquip={handleEquip} onUnequip={handleUnequip} equipped loading={loading} />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Borsa */}
      <div>
        <p className="text-surface-500 uppercase tracking-wider mb-1.5">
          Borsa <span className="text-surface-600 normal-case">({bag.length} oggetti)</span>
        </p>
        {bag.length === 0 ? (
          <p className="text-surface-600 text-center py-3">Borsa vuota</p>
        ) : (
          <div className="space-y-1.5">
            {bag.map((item) => (
              <ItemCard key={item.id} item={item} onEquip={handleEquip} onAppraise={handleAppraise} loading={loading} />
            ))}
          </div>
        )}
      </div>

      {/* Bonus totali da equip */}
      {inventory.stat_bonuses_from_equipment && (() => {
        const b = inventory.stat_bonuses_from_equipment
        const hasBonus = Object.values(b).some(v => typeof v === 'number' && v > 0)
        if (!hasBonus) return null
        return (
          <div className="border-t border-surface-700 pt-2">
            <p className="text-surface-500 uppercase tracking-wider mb-1.5">Bonus equipaggiamento</p>
            <div className="flex flex-wrap gap-1">
              {['STR','DEX','AGI','TEC','VIT','LUC'].map(s =>
                b[s] > 0 ? (
                  <span key={s} className="text-xs bg-surface-700 px-1.5 py-0.5 rounded font-mono text-accent">
                    +{b[s]} {STAT_LABELS[s]}
                  </span>
                ) : null
              )}
              {b.hp_bonus > 0 && <span className="text-xs bg-surface-700 px-1.5 py-0.5 rounded font-mono text-red-400">+{b.hp_bonus} HP</span>}
              {b.mp_bonus > 0 && <span className="text-xs bg-surface-700 px-1.5 py-0.5 rounded font-mono text-blue-400">+{b.mp_bonus} MP</span>}
            </div>
          </div>
        )
      })()}
    </div>
  )
}
