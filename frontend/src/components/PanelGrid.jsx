import { useState, useCallback, useRef, useEffect } from 'react'
import { STORAGE_PREFIX } from '../config.js'
import { ResponsiveGridLayout } from 'react-grid-layout'
import 'react-grid-layout/css/styles.css'
import 'react-resizable/css/styles.css'
import { PANELS, PRESETS } from './PanelRegistry.js'

const STORAGE_KEY = `${STORAGE_PREFIX}-layout`

function loadLayouts(charId) {
  try {
    const raw = localStorage.getItem(`${STORAGE_KEY}-${charId}`)
    return raw ? JSON.parse(raw) : null
  } catch { return null }
}

function saveLayouts(charId, layouts) {
  try {
    localStorage.setItem(`${STORAGE_KEY}-${charId}`, JSON.stringify(layouts))
  } catch {}
}

// Hook manuale per misurare la larghezza del container — versione semplice
// che funziona con react-grid-layout v2 (che richiede width esplicita).
function useContainerWidth(ref) {
  const [width, setWidth] = useState(1280)

  useEffect(() => {
    if (!ref.current) return
    const el = ref.current
    setWidth(el.getBoundingClientRect().width)

    const ro = new ResizeObserver(([entry]) => {
      setWidth(entry.contentRect.width)
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref])

  return width
}

export default function PanelGrid({ charId }) {
  const containerRef = useRef(null)
  const width = useContainerWidth(containerRef)

  const [layouts, setLayouts] = useState(() => loadLayouts(charId) ?? PRESETS.default)
  const [activePanels, setActivePanels] = useState(() => {
    const saved = loadLayouts(charId)
    const source = saved ?? PRESETS.default
    return source.lg.map((l) => l.i)
  })
  const [preset, setPreset] = useState('default')
  const [showControls, setShowControls] = useState(false)
  const [modified, setModified] = useState(false)
  const [savedFlash, setSavedFlash] = useState(false)

  const onLayoutChange = useCallback((_, allLayouts) => {
    setLayouts(allLayouts)
    setModified(true)
    // auto-salva comunque, ma mostra il pulsante "Salva" come conferma esplicita
  }, [])

  function saveCustomLayout() {
    saveLayouts(charId, layouts)
    setModified(false)
    setSavedFlash(true)
    setTimeout(() => setSavedFlash(false), 1500)
  }

  function applyPreset(name) {
    const newLayouts = PRESETS[name]
    if (!newLayouts) return
    setPreset(name)
    setLayouts(newLayouts)
    setActivePanels(newLayouts.lg.map((l) => l.i))
    saveLayouts(charId, newLayouts)
  }

  function togglePanel(id) {
    setActivePanels((prev) =>
      prev.includes(id) ? prev.filter((p) => p !== id) : [...prev, id]
    )
  }

  const visibleLayouts = {
    lg: (layouts.lg ?? PRESETS.default.lg).filter((l) => activePanels.includes(l.i)),
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-3 py-1.5 border-b border-surface-700 bg-surface-800 shrink-0 flex-wrap">
        <span className="text-surface-500 text-xs">Layout:</span>
        {Object.keys(PRESETS).map((name) => (
          <button
            key={name}
            onClick={() => applyPreset(name)}
            className={`text-xs px-2 py-0.5 rounded capitalize transition-colors ${
              preset === name
                ? 'bg-accent text-white'
                : 'text-surface-400 hover:text-surface-200 border border-surface-600'
            }`}
          >
            {name}
          </button>
        ))}
        <div className="h-3 w-px bg-surface-700 mx-1" />
        {modified && (
          <button
            onClick={saveCustomLayout}
            className={`text-xs px-2 py-0.5 rounded border transition-colors ${
              savedFlash
                ? 'border-green-600 text-green-400 bg-green-900/20'
                : 'border-accent text-accent hover:bg-accent/10 animate-pulse'
            }`}
          >
            {savedFlash ? '✓ Salvato' : 'Salva layout'}
          </button>
        )}
        <button
          onClick={() => setShowControls((v) => !v)}
          className="text-xs text-surface-400 hover:text-surface-200"
        >
          Panel ▾
        </button>
        {showControls && (
          <div className="flex items-center gap-1.5 flex-wrap">
            {Object.values(PANELS).map((p) => (
              <button
                key={p.id}
                onClick={() => togglePanel(p.id)}
                className={`text-xs px-2 py-0.5 rounded border transition-colors ${
                  activePanels.includes(p.id)
                    ? 'border-accent text-accent bg-accent/10'
                    : 'border-surface-600 text-surface-500'
                }`}
              >
                {p.icon} {p.label}
              </button>
            ))}
            <button
              onClick={() => applyPreset(preset)}
              className="text-xs text-surface-500 hover:text-surface-300 ml-1"
              title="Ripristina layout preset"
            >
              Reset
            </button>
          </div>
        )}
      </div>

      {/* Grid — ref misura la larghezza reale, passata esplicitamente al grid */}
      <div className="flex-1 overflow-auto min-h-0" ref={containerRef}>
        {width > 0 && (
          <ResponsiveGridLayout
            className="layout"
            layouts={visibleLayouts}
            width={width}
            breakpoints={{ lg: 0 }}
            cols={{ lg: 12 }}
            rowHeight={50}
            margin={[8, 8]}
            containerPadding={[8, 8]}
            onLayoutChange={onLayoutChange}
            dragConfig={{ enabled: true, handle: '.panel-drag-handle' }}
            resizeConfig={{ enabled: true, handles: ['s', 'e', 'se', 'sw', 'n', 'ne', 'nw', 'w'] }}
          >
            {activePanels
              .filter((id) => PANELS[id])
              .map((id) => {
                const PanelComponent = PANELS[id].component
                return (
                  <div key={id} className="flex flex-col bg-surface-800 border border-surface-700 rounded-lg overflow-hidden">
                    <div className="panel-drag-handle flex items-center gap-1.5 px-2 py-1 bg-surface-800 border-b border-surface-700 cursor-grab active:cursor-grabbing shrink-0 select-none">
                      <span className="text-surface-500 text-xs">⠿</span>
                      <span className="text-surface-400 text-xs">{PANELS[id].icon} {PANELS[id].label}</span>
                      <button
                        className="ml-auto text-surface-600 hover:text-surface-300 text-xs leading-none"
                        onMouseDown={(e) => e.stopPropagation()}
                        onClick={() => togglePanel(id)}
                      >
                        ×
                      </button>
                    </div>
                    <div className="flex-1 overflow-auto p-2">
                      <PanelComponent />
                    </div>
                  </div>
                )
              })}
          </ResponsiveGridLayout>
        )}
      </div>
    </div>
  )
}
