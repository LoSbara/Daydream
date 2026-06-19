import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client.js'

const TYPE_COLORS = {
  active:  'text-blue-400 border-blue-800',
  passive: 'text-green-400 border-green-800',
}

function CostBadge({ mpCost, stmCost, cooldown }) {
  const parts = []
  if (mpCost > 0)   parts.push(<span key="mp"  className="text-blue-400">MP:{mpCost}</span>)
  if (stmCost > 0)  parts.push(<span key="stm" className="text-yellow-400">STM:{stmCost}</span>)
  if (cooldown > 0) parts.push(<span key="cd"  className="text-surface-400">CD:{cooldown}t</span>)
  if (parts.length === 0) return null
  return <span className="flex gap-1.5">{parts}</span>
}

function SkillCard({ skill, onUnlock, unlocking }) {
  const isUnlocked  = skill.unlocked
  const isAvailable = skill.available
  const [expanded, setExpanded] = useState(false)

  const borderClass = isUnlocked
    ? 'border-accent bg-accent/5'
    : isAvailable
    ? 'border-surface-500 hover:border-surface-300 cursor-pointer'
    : 'border-surface-700 opacity-50'

  return (
    <div
      className={`rounded border p-2 text-xs transition-all ${borderClass}`}
      onClick={() => (isUnlocked || isAvailable) && setExpanded(v => !v)}
    >
      <div className="flex items-start gap-1.5">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className={`font-semibold ${isUnlocked ? 'text-accent' : 'text-surface-200'}`}>
              {skill.name}
            </span>
            <span className={`text-xs px-1 rounded border ${TYPE_COLORS[skill.type] ?? ''}`}>
              {skill.type === 'active' ? 'Attivo' : 'Passivo'}
            </span>
            {isUnlocked && <span className="text-accent text-xs">✓</span>}
          </div>
          <p className="text-surface-400 mt-0.5 leading-tight">{skill.description}</p>
        </div>
      </div>

      {expanded && (
        <div className="mt-2 pt-2 border-t border-surface-700 space-y-1.5">
          <p className="text-surface-300 leading-relaxed">{skill.effect_desc}</p>
          <div className="flex items-center gap-3 flex-wrap">
            <CostBadge mpCost={skill.mp_cost} stmCost={skill.stm_cost} cooldown={skill.cooldown} />
            {skill.stat_bonus && Object.keys(skill.stat_bonus).length > 0 && (
              <span className="text-green-400 font-mono">
                {Object.entries(skill.stat_bonus).map(([k, v]) => `+${v} ${k}`).join(' ')}
              </span>
            )}
          </div>
          {skill.spec_required && (
            <p className="text-purple-400 text-xs">Richiede specializzazione: {skill.spec_required}</p>
          )}
          {!isUnlocked && isAvailable && (
            <button
              onClick={(e) => { e.stopPropagation(); onUnlock(skill.id) }}
              disabled={unlocking === skill.id}
              className="w-full mt-1 py-1 px-2 rounded border border-accent text-accent hover:bg-accent/10 disabled:opacity-40 transition-colors"
            >
              {unlocking === skill.id ? 'Sbloccando…' : `🔓 Sblocca (${skill.cost ?? 3} pt)`}
            </button>
          )}
          {!isUnlocked && !isAvailable && skill.prerequisites?.length > 0 && (
            <p className="text-surface-600 text-xs">Prerequisiti mancanti</p>
          )}
        </div>
      )}
    </div>
  )
}

const UPGRADE_COSTS = [0, 15, 30, 50, 80]

export default function SkillTreePanel() {
  const character       = useGameStore((s) => s.character)
  const updateCharacter = useGameStore((s) => s.updateCharacter)
  const [data, setData]           = useState(null)
  const [loading, setLoading]     = useState(true)
  const [unlocking, setUnlocking] = useState(null)
  const [actLoading, setActLoading] = useState(null)
  const [error, setError]         = useState(null)

  function load() {
    setLoading(true)
    api.get('/character/skill-tree')
      .then(setData)
      .catch(() => setError("Impossibile caricare l'albero."))
      .finally(() => setLoading(false))
  }

  useEffect(() => { if (character) load() }, [character?.id])

  async function handleUnlock(skillId) {
    setUnlocking(skillId)
    setError(null)
    try {
      const res = await api.post('/character/skill-tree/unlock', { skill_id: skillId })
      updateCharacter(res)
      load()
    } catch (err) {
      setError(err.message || 'Errore sblocco')
    }
    setUnlocking(null)
  }

  async function handleUpgradeCustomSkill(skillName, currentLevel) {
    setActLoading(`upgrade_${skillName}`)
    setError(null)
    try {
      const res = await api.post('/character/custom-skill/upgrade', { skill_name: skillName })
      // res ha { skill, tree_points_available }
      setData(prev => prev ? {
        ...prev,
        points_available: res.tree_points_available,
      } : prev)
      // aggiorna il personaggio nello store con la skill potenziata
      updateCharacter({
        ...character,
        tree_points_available: res.tree_points_available,
        custom_skills: (character.custom_skills ?? []).map(s =>
          s.name === skillName ? res.skill : s
        ),
      })
    } catch (err) {
      setError(err.message || 'Errore potenziamento')
    }
    setActLoading(null)
  }

  if (loading) return <p className="text-surface-500 text-xs p-2">Caricamento…</p>
  if (error)   return <p className="text-red-400 text-xs p-2">{error}</p>
  if (!data)   return null

  const { tree, points_available } = data

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Header punti */}
      <div className="shrink-0 flex items-center justify-between px-2 py-1.5 border-b border-surface-700 bg-surface-800">
        <span className="text-surface-400 text-xs">Albero abilità — {tree.class}</span>
        <span className={`text-xs font-mono font-bold ${points_available > 0 ? 'text-accent animate-pulse' : 'text-surface-500'}`}>
          {points_available} punt{points_available === 1 ? 'o' : 'i'} skill
        </span>
      </div>

      {/* Rami */}
      <div className="flex-1 overflow-y-auto min-h-0 p-2 space-y-4">
        {tree.branches.map((branch) => (
          <div key={branch.name}>
            <div className="flex items-center gap-1.5 mb-2">
              <span className="text-base">{branch.icon}</span>
              <span className="text-surface-300 text-xs font-semibold uppercase tracking-wider">{branch.name}</span>
              <div className="flex-1 h-px bg-surface-700 ml-1" />
            </div>
            <div className="space-y-1.5 pl-1 border-l-2 border-surface-700">
              {branch.skills.map((skill) => (
                <SkillCard
                  key={skill.id}
                  skill={skill}
                  onUnlock={handleUnlock}
                  unlocking={unlocking}
                />
              ))}
            </div>
          </div>
        ))}

        {/* Abilità Uniche narrative */}
        {character?.custom_skills?.length > 0 && (
          <div className="mt-4">
            <div className="flex items-center gap-1.5 mb-2">
              <span className="text-base">✨</span>
              <span className="text-surface-300 text-xs font-semibold uppercase tracking-wider">Abilità Uniche</span>
              <div className="flex-1 h-px bg-surface-700 ml-1" />
            </div>
            <div className="space-y-1.5 pl-1 border-l-2 border-yellow-800">
              {character.custom_skills.map((skill) => {
                const currentLv = skill.level || 0
                const maxLv = skill.max_level || 5
                const canUpgrade = currentLv < maxLv
                const upgradeCost = UPGRADE_COSTS[currentLv] ?? 80
                const isUpgrading = actLoading === `upgrade_${skill.name}`
                return (
                  <div key={skill.id} className="rounded border border-yellow-800 bg-yellow-900/10 p-2 text-xs">
                    <div className="flex items-center gap-1.5 mb-0.5 flex-wrap">
                      <span className="text-yellow-400 font-semibold">
                        {skill.name} <span className="text-yellow-600 font-mono">(Lv{Math.max(currentLv, 1)}/{maxLv})</span>
                      </span>
                      <span className="text-yellow-600 border border-yellow-800 px-1 rounded">
                        {skill.type === 'active' ? 'Attivo' : 'Passivo'}
                      </span>
                    </div>
                    <p className="text-surface-400 leading-tight mb-1">{skill.description}</p>
                    <p className="text-surface-300 leading-tight">{skill.effect_desc}</p>
                    {skill.origin && (
                      <p className="text-yellow-700 mt-1 italic">Origine: {skill.origin}</p>
                    )}
                    {canUpgrade && currentLv > 0 && (
                      <button
                        onClick={() => handleUpgradeCustomSkill(skill.name, currentLv)}
                        disabled={isUpgrading}
                        className="mt-1.5 w-full py-0.5 px-2 rounded border border-yellow-700 text-yellow-400 hover:bg-yellow-900/30 disabled:opacity-40 transition-colors"
                      >
                        {isUpgrading ? 'Potenziando…' : `⬆ Potenzia (${upgradeCost} pt)`}
                      </button>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
