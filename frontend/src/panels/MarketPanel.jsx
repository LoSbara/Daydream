import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

const RARITY_COLORS = {
  common: 'text-surface-400', uncommon: 'text-green-400',
  rare: 'text-blue-400', epic: 'text-purple-400', legendary: 'text-yellow-400',
}
const RARITY_BORDER = {
  common: 'border-surface-600', uncommon: 'border-green-800',
  rare: 'border-blue-800', epic: 'border-purple-800', legendary: 'border-yellow-700',
}
const RARITY_LABELS = {
  common: 'Comune', uncommon: 'Non comune', rare: 'Raro',
  epic: 'Epico', legendary: 'Leggendario',
}
const TYPE_ICONS = {
  weapon: '⚔', armor: '🛡', offhand: '🔮', accessory: '💍',
  head: '⛑', legs: '🩱', boots: '👢', consumable: '🧪', material: '💎',
}

const TYPE_LABELS_IT = {
  weapon: 'Arma', armor: 'Armatura', offhand: 'Oggetto Secondario',
  accessory: 'Accessorio', head: 'Copricapo', legs: 'Gambali',
  boots: 'Calzari', consumable: 'Consumabile', material: 'Materiale',
}

function StatBadges({ statBonus, perceivedStatBonus, analyzed }) {
  // Item analizzato: mostra stat percepite con "~" (potrebbero essere sbagliate)
  if (analyzed && perceivedStatBonus && Object.keys(perceivedStatBonus).length > 0) {
    return (
      <div className="flex flex-wrap gap-1 mt-1">
        {Object.entries(perceivedStatBonus).map(([k, v]) => (
          <span key={k} className="text-xs bg-surface-700 px-1 py-0.5 rounded font-mono text-yellow-400/70">
            ~{v > 0 ? '+' : ''}{v} {k}
          </span>
        ))}
      </div>
    )
  }
  // Non analizzato: nessuna stat mostrata
  if (!analyzed) return null
  // Analizzato ma senza stat (es. item comune)
  if (!statBonus || Object.keys(statBonus).length === 0) return null
  return (
    <div className="flex flex-wrap gap-1 mt-1">
      {Object.entries(statBonus).map(([k, v]) => (
        <span key={k} className="text-xs bg-surface-700 px-1 py-0.5 rounded font-mono text-surface-300">
          +{v} {k}
        </span>
      ))}
    </div>
  )
}

export default function MarketPanel() {
  const session = useGameStore((s) => s.session)
  const character = useGameStore((s) => s.character)
  const setInventory = useGameStore((s) => s.setInventory)
  const updateCharacter = useGameStore((s) => s.updateCharacter)

  const [market, setMarket]            = useState(null)
  const [loading, setLoading]          = useState(false)
  const [actionLoading, setActLoading] = useState(null)
  const [negoResult, setNegoResult]    = useState(null) // { itemId, narrative, new_price, success }
  const [error, setError]              = useState(null)

  async function handleAnalyze(itemId) {
    setActLoading(`analyze_${itemId}`)
    try {
      const res = await api.post('/market/analyze', { item_id: itemId })
      setMarket(res)
    } catch {}
    setActLoading(null)
  }

  function fetchMarket() {
    setLoading(true)
    setError(null)
    api.get('/market/browse')
      .then(setMarket)
      .catch(() => setError('Impossibile caricare il mercato.'))
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchMarket() }, [session?.location])

  async function handleBuy(itemId) {
    setActLoading(`buy_${itemId}`)
    setError(null)
    try {
      const res = await api.post('/market/buy', { item_id: itemId })
      setMarket(res.market)
      setInventory(res.inventory)
      updateCharacter(res.character)
    } catch (err) {
      setError(err.message || 'Acquisto fallito')
    }
    setActLoading(null)
  }

  async function handleNegotiate(itemId) {
    setActLoading(`nego_${itemId}`)
    setNegoResult(null)
    setError(null)
    try {
      const res = await api.post('/market/negotiate', { item_id: itemId })
      setNegoResult({ itemId, ...res })
      // aggiorna il prezzo nel market locale
      setMarket(prev => prev ? {
        ...prev,
        listings: prev.listings.map(l =>
          l.item.id === itemId ? { ...l, price: res.new_price, nego_done: true } : l
        )
      } : prev)
    } catch (err) {
      setError(err.message || 'Contrattazione fallita')
    }
    setActLoading(null)
  }

  if (loading) return (
    <div className="flex items-center justify-center h-full text-surface-500 text-xs">
      Sfogliando il mercato…
    </div>
  )

  if (error) return (
    <div className="flex flex-col items-center justify-center h-full gap-2 text-center p-3">
      <p className="text-red-400 text-xs">{error}</p>
      <button onClick={fetchMarket} className="text-xs text-accent hover:underline">Riprova</button>
    </div>
  )

  if (!market) return (
    <div className="flex flex-col items-center justify-center h-full text-center p-4">
      <p className="text-3xl mb-2">🏪</p>
      <p className="text-surface-500 text-xs">Nessun mercato nelle vicinanze.</p>
      <button onClick={fetchMarket} className="mt-2 text-xs text-accent hover:underline">Cerca</button>
    </div>
  )

  const gold = character?.money ?? 0
  const available = market.listings?.filter(l => !l.sold) ?? []

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex items-center justify-between px-2 py-1.5 border-b border-surface-700 shrink-0">
        <span className="text-surface-400 text-xs">📍 {market.location || 'Mercato'}</span>
        <span className="text-yellow-400 text-xs font-mono">💰 {gold} oro</span>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0 p-2 space-y-2">
        {available.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-center">
            <p className="text-surface-500 text-xs">Tutto esaurito.</p>
            <button onClick={fetchMarket} className="mt-2 text-xs text-accent hover:underline">Aggiorna</button>
          </div>
        )}

        {available.map((listing) => {
          const item       = listing.item
          const analyzed   = listing.analyzed
          // Se non analizzato, mostriamo solo tipo generico e border neutro
          const displayRarity = analyzed ? (item.perceived_rarity || item.rarity) : null
          const color      = analyzed ? (RARITY_COLORS[displayRarity] ?? 'text-surface-400') : 'text-surface-500'
          const border     = analyzed ? (RARITY_BORDER[displayRarity] ?? 'border-surface-600') : 'border-surface-700'
          const icon       = TYPE_ICONS[item.type] ?? '📦'
          const isBuying   = actionLoading === `buy_${item.id}`
          const isNego     = actionLoading === `nego_${item.id}`
          const isAnalyzing = actionLoading === `analyze_${item.id}`
          const canAfford  = gold >= listing.price
          const thisNego   = negoResult?.itemId === item.id

          return (
            <div key={item.id} className={`rounded border p-2 ${border} bg-surface-900/40 text-xs`}>
              <div className="flex items-start gap-1.5">
                <span className="text-base leading-none mt-0.5">{icon}</span>
                <div className="flex-1 min-w-0">
                  <div className="flex items-baseline gap-1.5 flex-wrap">
                    {analyzed ? (
                      <>
                        <span className={`font-semibold ${color}`}>{item.name}</span>
                        <span className={`text-xs ${color} opacity-60`}>
                          {RARITY_LABELS[displayRarity] ?? displayRarity}
                          {item.perceived_rarity && item.perceived_rarity !== item.rarity && ' ~'}
                        </span>
                      </>
                    ) : (
                      <span className="font-semibold text-surface-500 italic">
                        {TYPE_LABELS_IT[item.type] ?? 'Oggetto'} Sconosciuto
                      </span>
                    )}
                  </div>

                  <StatBadges
                    statBonus={item.stat_bonus}
                    perceivedStatBonus={item.perceived_stat_bonus}
                    analyzed={analyzed}
                  />

                  {/* Nota analisi */}
                  {analyzed && (
                    <p className="text-surface-600 text-xs mt-1 italic">Analisi propria · qualità dipende da TEC</p>
                  )}

                  {/* Risultato contrattazione */}
                  {thisNego && (
                    <div className={`mt-1.5 p-1.5 rounded text-xs ${negoResult.success ? 'bg-green-900/20 border border-green-800 text-green-300' : 'bg-surface-800 border border-surface-700 text-surface-400'}`}>
                      {negoResult.narrative}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex items-center justify-between mt-2 gap-1.5">
                <div className="flex items-center gap-1">
                  {thisNego && negoResult.success && (
                    <span className="text-surface-500 line-through font-mono">{negoResult.old_price}</span>
                  )}
                  <span className={`font-mono font-bold ${canAfford ? 'text-yellow-400' : 'text-red-400'}`}>
                    💰 {listing.price}
                  </span>
                </div>
                <div className="flex gap-1">
                  {/* Analizza — solo se non ancora analizzato */}
                  {!analyzed && (
                    <button
                      onClick={() => handleAnalyze(item.id)}
                      disabled={!!actionLoading}
                      className="px-2 py-1 rounded border border-surface-600 text-surface-400 hover:border-yellow-600 hover:text-yellow-400 disabled:opacity-40 transition-colors"
                    >
                      {isAnalyzing ? '…' : '🔍 Analizza'}
                    </button>
                  )}
                  {!listing.nego_done && (
                    <button
                      onClick={() => handleNegotiate(item.id)}
                      disabled={!!actionLoading}
                      className="px-2 py-1 rounded border border-surface-600 text-surface-400 hover:border-surface-400 hover:text-surface-200 disabled:opacity-40 transition-colors"
                    >
                      {isNego ? '…' : '🤝 Contratta'}
                    </button>
                  )}
                  <button
                    onClick={() => handleBuy(item.id)}
                    disabled={!!actionLoading || !canAfford}
                    className={`px-2 py-1 rounded border transition-colors disabled:opacity-40 ${
                      canAfford
                        ? 'border-accent text-accent hover:bg-accent/10'
                        : 'border-surface-700 text-surface-600'
                    }`}
                  >
                    {isBuying ? '…' : 'Acquista'}
                  </button>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      <div className="shrink-0 p-2 border-t border-surface-700">
        <button
          onClick={fetchMarket}
          className="w-full text-xs text-surface-500 hover:text-surface-300 py-1"
        >
          ↺ Aggiorna offerte
        </button>
      </div>
    </div>
  )
}
