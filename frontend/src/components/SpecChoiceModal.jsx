import { useState } from 'react'
import { api } from '../api/client.js'
import { useGameStore } from '../store/gameStore.js'

export default function SpecChoiceModal({ specChoice, onClose }) {
  const updateCharacter = useGameStore((s) => s.updateCharacter)
  const [selected, setSelected]   = useState(null)
  const [loading, setLoading]     = useState(false)
  const [error, setError]         = useState(null)

  async function confirm() {
    if (!selected) return
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/character/spec-choice', { spec_id: selected })
      updateCharacter(res)
      onClose()
    } catch (err) {
      setError(err.message || 'Errore nella scelta')
    } finally {
      setLoading(false)
    }
  }

  const tierLabel = `Percorso Tier ${specChoice.tier} (livello ${specChoice.level})`

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4">
      <div className="card w-full max-w-lg space-y-5">
        <div>
          <p className="text-accent text-xs uppercase tracking-widest mb-1">{tierLabel}</p>
          <h2 className="text-surface-100 text-xl font-bold">Scegli la tua specializzazione</h2>
          <p className="text-surface-400 text-sm mt-1">
            Questa scelta è permanente e definisce il percorso del tuo personaggio.
          </p>
        </div>

        {error && <p className="text-red-400 text-sm">{error}</p>}

        <div className="space-y-3">
          {specChoice.options.map((opt) => (
            <button
              key={opt.id}
              onClick={() => setSelected(opt.id)}
              className={`w-full text-left p-4 rounded-lg border transition-all ${
                selected === opt.id
                  ? 'border-accent bg-accent/10 shadow-lg shadow-accent/10'
                  : 'border-surface-600 hover:border-surface-400 hover:bg-surface-700/50'
              }`}
            >
              <div className="flex items-start justify-between gap-2 mb-1.5">
                <span className="text-surface-100 font-semibold">{opt.name}</span>
                {opt.stat_bonus && (
                  <span className="text-accent text-xs font-mono whitespace-nowrap">
                    {Object.entries(opt.stat_bonus).map(([k, v]) => `+${v} ${k}`).join(' ')}
                  </span>
                )}
              </div>
              <p className="text-surface-300 text-sm mb-2">{opt.description}</p>
              <p className="text-surface-500 text-xs italic">"{opt.flavor}"</p>
              <div className="mt-2 pt-2 border-t border-surface-700">
                <p className="text-surface-400 text-xs">
                  <span className="text-yellow-400">Passivo: </span>{opt.passive_desc}
                </p>
              </div>
            </button>
          ))}
        </div>

        <div className="flex gap-2 pt-1">
          <button
            onClick={confirm}
            disabled={!selected || loading}
            className="btn flex-1 py-2.5"
          >
            {loading ? 'Conferma in corso…' : 'Conferma scelta'}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-surface-400 hover:text-surface-200 text-sm"
          >
            Più tardi
          </button>
        </div>
      </div>
    </div>
  )
}
