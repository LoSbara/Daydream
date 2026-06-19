import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client'

function ObjectiveRow({ obj }) {
  return (
    <div className="flex items-start gap-1.5 text-xs">
      <span className={`mt-0.5 shrink-0 ${obj.done ? 'text-green-400' : 'text-surface-500'}`}>
        {obj.done ? '✓' : '○'}
      </span>
      <div className="flex-1 min-w-0">
        <span className={obj.done ? 'text-surface-400 line-through' : 'text-surface-300'}>
          {obj.description}
        </span>
        {obj.required > 1 && (
          <div className="mt-0.5">
            <div className="h-1 rounded bg-surface-700 overflow-hidden">
              <div
                className="h-full rounded bg-accent transition-all"
                style={{ width: `${Math.round((obj.current / obj.required) * 100)}%` }}
              />
            </div>
            <span className="text-surface-600 text-xs">{obj.current}/{obj.required}</span>
          </div>
        )}
      </div>
    </div>
  )
}

function timeRemaining(quest, gameTime) {
  if (!gameTime || (!quest.deadline_day && !quest.deadline_hour)) return null
  const nowMin = gameTime.day * 1440 + gameTime.hour * 60 + (gameTime.minute || 0)
  const deadMin = quest.deadline_day * 1440 + quest.deadline_hour * 60
  const rem = deadMin - nowMin
  if (rem <= 0) return 'Scaduta'
  const days = Math.floor(rem / 1440)
  const hours = Math.floor((rem % 1440) / 60)
  if (days > 0) return `${days}g ${hours}h`
  return `${hours}h`
}

const URGENCY_COLOR = {
  low:      'text-surface-400',
  medium:   'text-yellow-400',
  high:     'text-orange-400',
  critical: 'text-red-400',
}

const DIFF_STARS = (d) => '★'.repeat(d || 1) + '☆'.repeat(5 - (d || 1))

function QuestCard({ quest, gameTime }) {
  const [expanded, setExpanded] = useState(quest.status === 'active')

  const statusColor = {
    active:    'text-accent border-accent/30 bg-accent/5',
    completed: 'text-green-400 border-green-800 bg-green-900/10',
    failed:    'text-red-400 border-red-900 bg-red-900/10',
    expired:   'text-surface-500 border-surface-700 bg-surface-800/50',
  }[quest.status] ?? ''

  const allDone = quest.objectives?.every(o => o.done)
  const remaining = quest.status === 'active' ? timeRemaining(quest, gameTime) : null
  const urgencyColor = URGENCY_COLOR[quest.urgency] ?? 'text-surface-400'

  return (
    <div className={`rounded border p-2.5 ${statusColor}`}>
      <button className="w-full text-left flex items-center justify-between gap-2" onClick={() => setExpanded(v => !v)}>
        <div className="flex items-center gap-2 min-w-0 flex-wrap">
          <span className="text-xs font-bold truncate">{quest.title}</span>
          {quest.difficulty > 0 && (
            <span className="text-yellow-500/60 text-xs shrink-0" title={`Difficoltà ${quest.difficulty}/5`}>
              {DIFF_STARS(quest.difficulty)}
            </span>
          )}
          {quest.status === 'active' && allDone && (
            <span className="text-yellow-400 text-xs shrink-0">★ Completabile</span>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {remaining && (
            <span className={`text-xs font-mono ${remaining === 'Scaduta' ? 'text-red-400' : urgencyColor}`}>
              ⏱ {remaining}
            </span>
          )}
          <span className="text-surface-500 text-xs">{expanded ? '▲' : '▼'}</span>
        </div>
      </button>

      {expanded && (
        <div className="mt-2 space-y-2">
          <div className="flex items-center gap-2 flex-wrap">
            {quest.giver_npc && <p className="text-surface-500 text-xs italic">Da: {quest.giver_npc}</p>}
            {quest.urgency && (
              <span className={`text-xs ${urgencyColor}`}>[{quest.urgency}]</span>
            )}
          </div>

          <p className="text-surface-400 text-xs leading-relaxed">{quest.description}</p>

          {quest.escalation_stage > 0 && quest.escalations?.[quest.escalation_stage - 1] && (
            <div className="bg-orange-900/20 border border-orange-800/40 rounded p-1.5 text-xs text-orange-300">
              ⚠ {quest.escalations[quest.escalation_stage - 1].description}
            </div>
          )}

          {quest.objectives?.length > 0 && (
            <div className="space-y-1 border-t border-current/20 pt-2">
              {quest.objectives.map((obj, i) => <ObjectiveRow key={i} obj={obj} />)}
            </div>
          )}

          {(quest.rewards?.gold > 0 || quest.rewards?.exp > 0 || quest.rewards?.items?.length > 0) && (
            <div className="border-t border-current/20 pt-2 space-y-1">
              <div className="flex gap-3 text-xs flex-wrap">
                {quest.rewards.gold > 0 && <span className="text-yellow-400">💰 {quest.rewards.gold}g</span>}
                {quest.rewards.exp > 0  && <span className="text-blue-400">⭐ {quest.rewards.exp} EXP</span>}
              </div>
              {quest.rewards.items?.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {quest.rewards.items.map((item, i) => (
                    <span key={i} className="text-xs bg-surface-700 px-1.5 py-0.5 rounded text-surface-300">
                      🎁 {item.name || 'Item'}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function QuestPanel() {
  const { session } = useGameStore()
  const gameTime = session?.game_time
  const [quests, setQuests] = useState({ active: [], completed: [], failed: [] })
  const [loading, setLoading] = useState(true)
  const [showCompleted, setShowCompleted] = useState(false)

  useEffect(() => {
    let cancelled = false
    api.get('/quests')
      .then((res) => { if (!cancelled) { setQuests(res); setLoading(false) } })
      .catch(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [session?.turn_id])

  if (loading) {
    return <div className="text-surface-500 text-xs py-4 text-center">Caricamento quest…</div>
  }

  const active = quests.active ?? []
  const completed = quests.completed ?? []
  const failed = quests.failed ?? []

  return (
    <div className="space-y-3">
      {active.length === 0 ? (
        <p className="text-surface-600 text-xs italic text-center py-4">
          Nessuna quest attiva. Parla con un NPC o esplora il mondo.
        </p>
      ) : (
        <div className="space-y-2">
          <p className="text-surface-500 text-xs uppercase tracking-wider">
            Attive ({active.length})
          </p>
          {active.map((q) => <QuestCard key={q.id} quest={q} gameTime={gameTime} />)}
        </div>
      )}

      {(completed.length > 0 || failed.length > 0) && (
        <div className="border-t border-surface-700 pt-2">
          <button
            className="text-surface-500 text-xs hover:text-surface-300 transition-colors"
            onClick={() => setShowCompleted((v) => !v)}
          >
            {showCompleted ? '▲' : '▼'} Cronologia ({completed.length + failed.length})
          </button>
          {showCompleted && (
            <div className="mt-2 space-y-1.5">
              {[...completed, ...failed].map((q) => <QuestCard key={q.id} quest={q} gameTime={gameTime} />)}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
