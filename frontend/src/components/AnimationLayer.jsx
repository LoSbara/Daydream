import { useEffect, useRef, useState } from 'react'
import { useGameStore } from '../store/gameStore.js'

// Reacts to uiEvents emitted by the GM in each turn's DonePayload.
// uiEvents è un array di stringhe: ["SCREEN_SHAKE", "RED_FLASH", "OVERDRIVE", ...]
export default function AnimationLayer() {
  const uiEvents = useGameStore((s) => s.uiEvents)
  const overdrive = useGameStore((s) => s.overdrive)
  const [shake, setShake] = useState(false)
  const [redFlash, setRedFlash] = useState(false)
  const [death, setDeath] = useState(false)
  const [overdrivePulse, setOverdrivePulse] = useState(false)
  const prevEvents = useRef([])

  useEffect(() => {
    // Esegui effetti solo se uiEvents è cambiato
    if (uiEvents === prevEvents.current) return
    prevEvents.current = uiEvents

    if (!uiEvents?.length) return

    if (uiEvents.includes('SCREEN_SHAKE')) {
      setShake(true)
      setTimeout(() => setShake(false), 600)
    }
    if (uiEvents.includes('RED_FLASH')) {
      setRedFlash(true)
      setTimeout(() => setRedFlash(false), 400)
    }
    if (uiEvents.includes('DEATH')) {
      setDeath(true)
      // DEATH non si auto-dismissisce: il giocatore deve cliccare
    }
    if (uiEvents.includes('OVERDRIVE')) {
      setOverdrivePulse(true)
      setTimeout(() => setOverdrivePulse(false), 1000)
    }
  }, [uiEvents])

  return (
    <>
      {/* Screen shake: applica una classe CSS sull'elemento root del layout */}
      {shake && (
        <style>{`#game-root { animation: screenShake 0.5s cubic-bezier(.36,.07,.19,.97) both; }`}</style>
      )}

      {/* Red flash overlay */}
      {redFlash && (
        <div
          className="fixed inset-0 z-50 pointer-events-none"
          style={{ background: 'rgba(220,40,40,0.35)', animation: 'fadeOut 0.4s ease-out forwards' }}
        />
      )}

      {/* Overdrive pulse border */}
      {overdrivePulse && (
        <div
          className="fixed inset-0 z-40 pointer-events-none rounded"
          style={{ boxShadow: 'inset 0 0 60px 20px rgba(251,146,60,0.5)', animation: 'fadeOut 1s ease-out forwards' }}
        />
      )}

      {/* Overdrive permanente (mentre attivo) */}
      {overdrive && !overdrivePulse && (
        <div
          className="fixed inset-0 z-30 pointer-events-none"
          style={{ boxShadow: 'inset 0 0 30px 8px rgba(251,146,60,0.2)', animation: 'overdrivePulse 2s ease-in-out infinite' }}
        />
      )}

      {/* Death overlay */}
      {death && (
        <div className="fixed inset-0 z-60 flex flex-col items-center justify-center bg-black/85 backdrop-blur-sm">
          <div className="text-center space-y-4">
            <p className="text-red-500 text-6xl font-mono font-bold tracking-widest">MORTE</p>
            <p className="text-surface-300 text-sm">Il tuo personaggio è caduto in battaglia.</p>
            <p className="text-surface-400 text-xs">Continua la conversazione con il GM per il respawn.</p>
            <button
              className="mt-4 px-6 py-2 border border-red-800 text-red-400 rounded hover:bg-red-900/20 transition-colors text-sm"
              onClick={() => setDeath(false)}
            >
              Continua
            </button>
          </div>
        </div>
      )}
    </>
  )
}
