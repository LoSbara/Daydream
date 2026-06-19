import { useCallback } from 'react'
import { useGameStore } from '../store/gameStore'
import { api } from '../api/client'

export function useSkills() {
  const { skills, skillSlots, setSkills, updateLoadout, session } = useGameStore()

  // Carica le skill dal backend
  const loadSkills = useCallback(async () => {
    try {
      const res = await api.get('/skills')
      setSkills(res.skills, res.skill_slots)
    } catch {
      // Silenzioso: se fallisce rimane []
    }
  }, [setSkills])

  // Aggiorna il loadout (toggle in/out)
  const toggleLoadout = useCallback(async (skillId) => {
    const currentLoadout = session?.skill_loadout ?? []
    let newLoadout

    if (currentLoadout.includes(skillId)) {
      newLoadout = currentLoadout.filter((id) => id !== skillId)
    } else {
      if (currentLoadout.length >= skillSlots) return // loadout pieno
      newLoadout = [...currentLoadout, skillId]
    }

    try {
      await api.put('/character/loadout', { loadout: newLoadout })
      updateLoadout(newLoadout)
    } catch {
      // Non aggiornare lo store se il server ha rigettato
    }
  }, [session, skillSlots, updateLoadout])

  return { skills, skillSlots, loadSkills, toggleLoadout }
}
