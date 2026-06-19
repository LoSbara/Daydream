import { useGameStore } from '../store/gameStore'

const RARITY_CONFIG = {
  common:    { label: 'Comune',       color: 'text-surface-300',  border: 'border-surface-600' },
  uncommon:  { label: 'Non Comune',   color: 'text-green-400',    border: 'border-green-700' },
  rare:      { label: 'Rara',         color: 'text-blue-400',     border: 'border-blue-700' },
  epic:      { label: 'Epica',        color: 'text-purple-400',   border: 'border-purple-700' },
  legendary: { label: 'Leggendaria',  color: 'text-yellow-400',   border: 'border-yellow-600' },
}

const TYPE_ICONS = {
  weapon: '⚔',
  armor: '🛡',
  consumable: '🧪',
  material: '💎',
}

export default function LootPopup() {
  const { lootInfo, clearLootInfo } = useGameStore()

  if (!lootInfo) return null

  const hasItems = lootInfo.items?.length > 0

  return (
    // Overlay backdrop
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={clearLootInfo}
    >
      <div
        className="bg-surface-800 border border-surface-600 rounded-lg p-5 max-w-sm w-full mx-4 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="text-center mb-4">
          <p className="text-yellow-400 text-2xl mb-1">🎁</p>
          <h2 className="text-surface-100 font-bold text-lg">Loot ottenuto!</h2>
        </div>

        {/* Gold */}
        {lootInfo.gold > 0 && (
          <div className="flex items-center justify-between bg-yellow-900/20 border border-yellow-800/40 rounded p-2.5 mb-3">
            <span className="text-yellow-300 text-sm">💰 Gold</span>
            <span className="text-yellow-400 font-mono font-bold text-lg">+{lootInfo.gold.toLocaleString()}</span>
          </div>
        )}

        {/* Items */}
        {hasItems && (
          <div className="space-y-2 mb-4">
            {lootInfo.items.map((item, i) => {
              const cfg = RARITY_CONFIG[item.rarity] ?? RARITY_CONFIG.common
              return (
                <div
                  key={item.id ?? i}
                  className={`flex items-start gap-2 border rounded p-2.5 ${cfg.border} bg-surface-700/40`}
                >
                  <span className="text-lg shrink-0">{TYPE_ICONS[item.type] ?? '📦'}</span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-1.5">
                      <span className={`text-sm font-medium ${cfg.color}`}>{item.name}</span>
                      <span className={`text-xs ${cfg.color} opacity-60`}>[{cfg.label}]</span>
                    </div>
                    {item.appraised === false && item.rarity && item.rarity !== 'common' ? (
                      <div className="flex flex-wrap gap-1 mt-1">
                        <span className="text-xs bg-surface-800 border border-surface-600 px-1.5 py-0.5 rounded font-mono text-surface-600 select-none" style={{ filter: 'blur(3px)' }}>+? ???</span>
                        <span className="text-xs bg-surface-800 border border-surface-600 px-1.5 py-0.5 rounded font-mono text-surface-600 select-none" style={{ filter: 'blur(3px)' }}>+? ???</span>
                        <span className="text-xs text-yellow-600 mt-0.5 w-full">Non identificato</span>
                      </div>
                    ) : item.appraised && item.perceived_stat_bonus && Object.keys(item.perceived_stat_bonus).length > 0 ? (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {Object.entries(item.perceived_stat_bonus).map(([k, v]) => (
                          <span key={k} className="text-xs bg-surface-800 border border-surface-600 px-1.5 py-0.5 rounded font-mono text-yellow-400/70">
                            ~{v > 0 ? '+' : ''}{v} {k}
                          </span>
                        ))}
                      </div>
                    ) : item.stat_bonus && Object.keys(item.stat_bonus).length > 0 ? (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {Object.entries(item.stat_bonus).map(([k, v]) => (
                          <span key={k} className="text-xs bg-surface-800 border border-surface-600 px-1.5 py-0.5 rounded font-mono">
                            +{v} {k}
                          </span>
                        ))}
                      </div>
                    ) : null}
                  </div>
                  {item.price > 0 && (
                    <span className="text-yellow-500 text-xs font-mono shrink-0">{item.price}g</span>
                  )}
                </div>
              )
            })}
          </div>
        )}

        {/* Nota borsa */}
        {hasItems && (
          <p className="text-surface-500 text-xs text-center mb-4">
            Gli item sono stati aggiunti alla tua borsa.
          </p>
        )}

        <button
          className="btn w-full py-2"
          onClick={clearLootInfo}
        >
          Continua
        </button>
      </div>
    </div>
  )
}
