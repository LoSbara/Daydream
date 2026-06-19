import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore.js'
import { useGameStore } from '../store/gameStore.js'
import { api } from '../api/client.js'
import { APP_NAME } from '../config.js'
import LootPopup from '../panels/LootPopup.jsx'
import SpecChoiceModal from '../components/SpecChoiceModal.jsx'
import { useSkills } from '../hooks/useSkills.js'
import { useWebSocket } from '../hooks/useWebSocket.js'
import PanelGrid from '../components/PanelGrid.jsx'
import AnimationLayer from '../components/AnimationLayer.jsx'

const VALID_JOBS = ['Mercenario', 'Scout', 'Mago', 'Sacerdote', 'Ingegnere']

const JOB_DESCRIPTIONS = {
  Mercenario: 'Combattente versatile. Eccelle con le armi fisiche, alta resistenza.',
  Scout:      'Veloce e preciso. Specialista in esplorazione e attacchi furtivi.',
  Mago:       'Padrone dell\'arcano. Potenti incantesimi, fragile in mischia.',
  Sacerdote:  'Guaritore e supporto. Alto VIT, abilità di recupero HP.',
  Ingegnere:  'Tattico e inventore. Trappole, costrutti meccanici, versatilità tecnica.',
}

// Bonus classe: +3 a 2 stat (narrativo, applicato sopra l'allocazione)
const JOB_STAT_BONUSES = {
  Mercenario: { STR: 3, VIT: 3 },
  Scout:      { DEX: 3, AGI: 3 },
  Mago:       { TEC: 3, LUC: 3 },
  Sacerdote:  { VIT: 3, TEC: 3 },
  Ingegnere:  { TEC: 3, DEX: 3 },
}

const STAT_DESCS = {
  STR: { name: 'STR', full: 'Forza',     desc: 'Danno fisico in mischia, capacità di trasporto' },
  DEX: { name: 'DEX', full: 'Destrezza', desc: 'Precisione attacchi, danno ranged, critico' },
  AGI: { name: 'AGI', full: 'Agilità',   desc: 'Velocità, schivata, iniziativa in combattimento' },
  TEC: { name: 'TEC', full: 'Tecnica',   desc: 'Potere magico, identificazione oggetti, negoziazione' },
  VIT: { name: 'VIT', full: 'Vitalità',  desc: 'HP massimi, resistenza ai danni' },
  LUC: { name: 'LUC', full: 'Fortuna',   desc: 'Drop rate, rarità loot, fortuna in negoziazione' },
}

const STAT_KEYS = ['STR', 'DEX', 'AGI', 'TEC', 'VIT', 'LUC']
const POOL_TOTAL = 15
const STAT_BASE  = 5

function initAllocStats() {
  return { STR: STAT_BASE, DEX: STAT_BASE, AGI: STAT_BASE, TEC: STAT_BASE, VIT: STAT_BASE, LUC: STAT_BASE }
}

function CharacterCreation({ onCreated }) {
  const [step, setStep]   = useState(1) // 1 = nome+classe, 2 = stat
  const [name, setName]   = useState('')
  const [job, setJob]     = useState('Mercenario')
  const [stats, setStats] = useState(initAllocStats)
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState(null)

  const poolUsed = STAT_KEYS.reduce((sum, k) => sum + (stats[k] - STAT_BASE), 0)
  const poolLeft = POOL_TOTAL - poolUsed

  function adjustStat(attr, delta) {
    setStats(prev => {
      const next = prev[attr] + delta
      if (next < STAT_BASE) return prev
      if (delta > 0 && poolLeft <= 0) return prev
      return { ...prev, [attr]: next }
    })
  }

  async function handleCreate() {
    setLoading(true)
    setError(null)
    try {
      const res = await api.post('/character', { name: name.trim(), job, initial_stats: stats })
      onCreated(res)
    } catch (err) {
      setError(err.message || 'Errore nella creazione del personaggio')
      setStep(2) // rimane allo step 2 per mostrare l'errore
    } finally {
      setLoading(false)
    }
  }

  const jobBonuses = JOB_STAT_BONUSES[job] ?? {}

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface p-4">
      <div className="card w-full max-w-md space-y-6">

        {/* Header */}
        <div className="text-center">
          <p className="text-4xl mb-2">⚔</p>
          <h1 className="text-surface-100 text-xl font-bold">Crea il tuo personaggio</h1>
          <p className="text-surface-400 text-sm mt-1">
            {step === 1 ? `Il tuo avatar in ${APP_NAME}. Scegli con cura.` : 'Distribuisci i tuoi punti stat iniziali.'}
          </p>
          {/* Indicatore step */}
          <div className="flex items-center justify-center gap-2 mt-3">
            {[1, 2].map(s => (
              <div key={s} className={`h-1.5 w-10 rounded-full transition-colors ${s <= step ? 'bg-accent' : 'bg-surface-600'}`} />
            ))}
          </div>
        </div>

        {error && (
          <div className="bg-red-900/20 border border-red-800 rounded p-3 text-red-400 text-sm">
            {error}
          </div>
        )}

        {/* STEP 1: nome + classe */}
        {step === 1 && (
          <div className="space-y-4">
            <div>
              <label className="block text-surface-300 text-sm mb-1" htmlFor="charName">
                Nome personaggio
              </label>
              <input
                id="charName"
                type="text"
                className="input w-full"
                placeholder={`Il tuo nome in ${APP_NAME}…`}
                value={name}
                onChange={(e) => setName(e.target.value)}
                minLength={2}
                maxLength={24}
              />
            </div>

            <div>
              <label className="block text-surface-300 text-sm mb-2">Classe</label>
              <div className="space-y-2">
                {VALID_JOBS.map((j) => (
                  <label
                    key={j}
                    className={`flex items-start gap-3 p-3 rounded border cursor-pointer transition-colors ${
                      job === j
                        ? 'border-accent bg-accent/10'
                        : 'border-surface-600 hover:border-surface-400'
                    }`}
                  >
                    <input
                      type="radio"
                      name="job"
                      value={j}
                      checked={job === j}
                      onChange={() => setJob(j)}
                      className="mt-0.5"
                    />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <p className="text-surface-100 text-sm font-medium">{j}</p>
                        {Object.entries(JOB_STAT_BONUSES[j] ?? {}).map(([s, v]) => (
                          <span key={s} className="text-xs px-1 py-0.5 rounded bg-accent/20 text-accent font-mono">+{v} {s}</span>
                        ))}
                      </div>
                      <p className="text-surface-400 text-xs mt-0.5">{JOB_DESCRIPTIONS[j]}</p>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            <button
              onClick={() => setStep(2)}
              disabled={!name.trim() || name.trim().length < 2}
              className="btn w-full py-2.5"
            >
              Avanti — Alloca le stat
            </button>
          </div>
        )}

        {/* STEP 2: allocazione stat */}
        {step === 2 && (
          <div className="space-y-4">
            {/* Pool rimanente */}
            <div className="flex items-center justify-between px-3 py-2 rounded bg-surface-800 border border-surface-600">
              <span className="text-surface-300 text-sm">Punti disponibili</span>
              <span className={`text-xl font-mono font-bold ${poolLeft === 0 ? 'text-green-400' : 'text-accent'}`}>
                {poolLeft}
              </span>
            </div>

            <div className="space-y-2">
              {STAT_KEYS.map(attr => {
                const info    = STAT_DESCS[attr]
                const bonus   = jobBonuses[attr] ?? 0
                const current = stats[attr]
                const total   = current + bonus
                return (
                  <div key={attr} className="rounded border border-surface-700 bg-surface-900/50 p-2.5">
                    <div className="flex items-center justify-between mb-1">
                      <div className="flex items-center gap-2">
                        <span className="text-surface-100 text-sm font-medium font-mono w-8">{info.name}</span>
                        <span className="text-surface-500 text-xs">{info.full}</span>
                        {bonus > 0 && (
                          <span className="text-xs px-1 py-0.5 rounded bg-accent/20 text-accent font-mono">+{bonus} classe</span>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => adjustStat(attr, -1)}
                          disabled={current <= STAT_BASE}
                          className="w-6 h-6 rounded border border-surface-600 text-surface-400 hover:border-surface-400 hover:text-surface-200 disabled:opacity-30 disabled:cursor-not-allowed text-sm leading-none flex items-center justify-center"
                        >
                          −
                        </button>
                        <span className="text-surface-100 font-mono text-sm w-10 text-center">
                          {current}
                          {bonus > 0 && <span className="text-accent"> ({total})</span>}
                        </span>
                        <button
                          onClick={() => adjustStat(attr, +1)}
                          disabled={poolLeft <= 0}
                          className="w-6 h-6 rounded border border-surface-600 text-surface-400 hover:border-accent hover:text-accent disabled:opacity-30 disabled:cursor-not-allowed text-sm leading-none flex items-center justify-center"
                        >
                          +
                        </button>
                      </div>
                    </div>
                    <p className="text-surface-600 text-xs">{info.desc}</p>
                  </div>
                )
              })}
            </div>

            <div className="flex gap-2">
              <button
                onClick={() => setStep(1)}
                className="px-4 py-2 rounded border border-surface-600 text-surface-400 hover:text-surface-200 hover:border-surface-400 text-sm transition-colors"
              >
                Indietro
              </button>
              <button
                onClick={handleCreate}
                disabled={loading}
                className="btn flex-1 py-2.5"
              >
                {loading ? 'Creazione in corso…' : `Crea personaggio`}
              </button>
            </div>

            {poolLeft > 0 && (
              <p className="text-surface-500 text-xs text-center">
                Hai ancora {poolLeft} punt{poolLeft === 1 ? 'o' : 'i'} da distribuire — puoi comunque procedere.
              </p>
            )}
          </div>
        )}

      </div>
    </div>
  )
}

const STAT_LABELS = {
  STR: { label: 'FOR', desc: 'Forza fisica, danno melee' },
  DEX: { label: 'DES', desc: 'Precisione, danno ranged' },
  AGI: { label: 'AGI', desc: 'Velocità, schivata' },
  TEC: { label: 'TEC', desc: 'Potere magico, MP max' },
  VIT: { label: 'VIT', desc: 'Resistenza, HP max' },
  LUC: { label: 'LUC', desc: 'Critico, drop rate' },
}

function StatAllocModal({ onClose }) {
  const character = useGameStore((s) => s.character)
  const updateCharacter = useGameStore((s) => s.updateCharacter)
  const [loading, setLoading] = useState(null)
  const [error, setError] = useState(null)
  const pts = character?.stat_points_available ?? 0

  if (!character) return null

  async function allocate(attr) {
    if (pts <= 0) return
    setLoading(attr)
    setError(null)
    try {
      const res = await api.put('/character/stats', { attribute: attr, points: 1 })
      updateCharacter(res)
      if ((res.stat_points_available ?? 0) === 0) onClose()
    } catch (err) {
      setError(err.message || 'Errore allocazione')
    } finally {
      setLoading(null)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="card w-full max-w-sm space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-surface-100 font-bold">Punti Stat Disponibili</h2>
            <p className="text-accent text-2xl font-mono font-bold">{pts}</p>
          </div>
          {pts === 0 && (
            <button onClick={onClose} className="text-surface-400 hover:text-surface-200 text-xl leading-none">×</button>
          )}
        </div>
        <p className="text-surface-400 text-xs">
          Alloca i punti per potenziare il tuo personaggio. VIT aumenta gli HP max, TEC aumenta gli MP max.
        </p>
        {error && (
          <p className="text-red-400 text-xs">{error}</p>
        )}
        <div className="space-y-2">
          {Object.entries(STAT_LABELS).map(([attr, info]) => {
            const current = character?.stats?.[attr] ?? 0
            return (
              <button
                key={attr}
                onClick={() => allocate(attr)}
                disabled={pts <= 0 || loading === attr}
                className="w-full flex items-center justify-between px-3 py-2 rounded border border-surface-600 hover:border-accent hover:bg-accent/5 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <div className="text-left">
                  <span className="text-surface-100 text-sm font-medium">{info.label}</span>
                  <span className="text-surface-400 text-xs ml-2">{info.desc}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-surface-300 font-mono text-sm">{current}</span>
                  {loading === attr ? (
                    <span className="text-surface-400 text-xs">…</span>
                  ) : (
                    <span className="text-accent text-sm">+1</span>
                  )}
                </div>
              </button>
            )
          })}
        </div>
        {pts > 0 && (
          <p className="text-surface-500 text-xs text-center">
            Puoi chiudere questa finestra e allocare i punti più tardi dalla scheda Stats.
          </p>
        )}
        <button onClick={onClose} className="text-surface-400 hover:text-surface-200 text-xs w-full text-center">
          Chiudi
        </button>
      </div>
    </div>
  )
}

function AnnouncementToasts() {
  const announcements = useGameStore((s) => s.announcements)
  const dismissAnnouncement = useGameStore((s) => s.dismissAnnouncement)

  if (!announcements.length) return null

  const levelColors = {
    info: 'border-surface-600 bg-surface-800',
    warning: 'border-yellow-600 bg-yellow-900/20',
    special: 'border-accent bg-accent/10',
  }

  return (
    <div className="fixed bottom-4 right-4 z-40 space-y-2 max-w-xs">
      {announcements.map((ann) => (
        <div
          key={ann.id}
          className={`flex items-start gap-2 px-3 py-2 rounded border text-sm shadow-lg ${levelColors[ann.level] ?? levelColors.info}`}
        >
          <span className="text-surface-200 flex-1">{ann.text}</span>
          <button
            onClick={() => dismissAnnouncement(ann.id)}
            className="text-surface-400 hover:text-surface-200 leading-none shrink-0"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  )
}

function GameClock() {
  const { session } = useGameStore()
  const gt = session?.game_time
  if (!gt) return null

  const isNight = gt.hour >= 21 || gt.hour < 6
  const icon = isNight ? '🌙' : gt.hour < 12 ? '🌅' : gt.hour < 17 ? '☀️' : '🌆'
  const hh = String(gt.hour).padStart(2, '0')
  const mm = String(gt.minute).padStart(2, '0')

  return (
    <span className="text-surface-400 text-xs font-mono tabular-nums select-none">
      {icon} Giorno {gt.day} {hh}:{mm}
    </span>
  )
}

function GameLayout() {
  const { user, clearAuth } = useAuthStore()
  const { character, session, lootInfo } = useGameStore()
  const specChoice = useGameStore((s) => s.specChoice)
  const setSpecChoice = useGameStore((s) => s.setSpecChoice)
  const navigate = useNavigate()
  const [showStatAlloc, setShowStatAlloc] = useState(false)

  // Connessione WebSocket per annunci globali
  useWebSocket()

  function handleLogout() {
    clearAuth()
    navigate('/login')
  }

  const hasStatPoints = (character?.stat_points_available ?? 0) > 0

  return (
    <div className="h-screen flex flex-col bg-surface overflow-hidden">
      {/* Topbar */}
      <header className="flex items-center justify-between px-4 py-2 border-b border-surface-700 bg-surface-800 shrink-0">
        <div className="flex items-center gap-3">
          <span className="font-mono font-bold text-accent tracking-tight text-lg">
            Day<span className="text-surface-100">dream</span>
          </span>
          {character && (
            <span className="text-surface-400 text-xs border-l border-surface-600 pl-3">
              {character.name} · Lv.{character.level} {character.job}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {hasStatPoints && (
            <button
              onClick={() => setShowStatAlloc(true)}
              className="text-xs px-2 py-0.5 rounded border border-yellow-600 text-yellow-400 hover:bg-yellow-900/20 transition-colors animate-pulse"
            >
              +{character?.stat_points_available ?? 0} punti stat
            </button>
          )}
          {specChoice && (
            <button
              onClick={() => {}}
              className="text-xs px-2 py-0.5 rounded border border-purple-600 text-purple-400 hover:bg-purple-900/20 transition-colors animate-pulse"
            >
              Nuova specializzazione!
            </button>
          )}
          {session?.combat_active && (
            <span className="text-red-400 text-xs font-mono animate-pulse">⚔ COMBATTIMENTO</span>
          )}
          <GameClock />
          <span className="text-surface-400 text-xs">{user?.username}</span>
          <button className="text-surface-400 hover:text-surface-200 text-xs" onClick={handleLogout}>
            Esci
          </button>
        </div>
      </header>

      {/* Layout principale — Dashboard Builder */}
      <div id="game-root" className="flex-1 overflow-hidden min-h-0">
        <PanelGrid charId={character?.id ?? 'guest'} />
      </div>

      {/* Overlay: loot */}
      {lootInfo && <LootPopup />}

      {/* Overlay: spec choice */}
      {specChoice && (
        <SpecChoiceModal specChoice={specChoice} onClose={() => setSpecChoice(null)} />
      )}

      {/* Overlay: stat allocation */}
      {showStatAlloc && <StatAllocModal onClose={() => setShowStatAlloc(false)} />}

      {/* Toast annunci WS */}
      <AnnouncementToasts />

      {/* Effetti animazione (SCREEN_SHAKE, RED_FLASH, DEATH, ecc.) */}
      <AnimationLayer />
    </div>
  )
}

export default function Game() {
  const navigate = useNavigate()
  const { clearAuth } = useAuthStore()
  const { setGameState, clearGameState } = useGameStore()
  const { loadSkills } = useSkills()
  const [loadingState, setLoadingState] = useState('loading')

  useEffect(() => {
    let cancelled = false

    async function init() {
      try {
        await api.get('/auth/me')
      } catch {
        clearAuth()
        navigate('/login')
        return
      }

      try {
        const state = await api.get('/state')
        if (cancelled) return
        setGameState(state.character, state.inventory, state.session, state.spec_choice ?? null)
        setLoadingState('ready')
        // Carica le skill dopo aver caricato lo stato
        loadSkills()
      } catch (err) {
        if (cancelled) return
        if (err.status === 404) {
          clearGameState()
          setLoadingState('no_character')
        } else {
          setLoadingState('error')
        }
      }
    }

    init()
    return () => { cancelled = true }
  }, []) // eslint-disable-line

  function handleCharacterCreated(data) {
    setGameState(data.character, data.inventory, data.session)
    setLoadingState('ready')
    loadSkills()
  }

  if (loadingState === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface">
        <div className="text-center">
          <p className="text-4xl mb-3 animate-spin">∞</p>
          <p className="text-surface-400 text-sm">Caricamento sessione…</p>
        </div>
      </div>
    )
  }

  if (loadingState === 'error') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface">
        <div className="card text-center max-w-sm">
          <p className="text-red-400 mb-3">Errore caricamento sessione</p>
          <button className="btn" onClick={() => window.location.reload()}>Riprova</button>
        </div>
      </div>
    )
  }

  if (loadingState === 'no_character') {
    return <CharacterCreation onCreated={handleCharacterCreated} />
  }

  return <GameLayout />
}
