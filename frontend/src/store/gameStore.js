import { create } from 'zustand'

export const useGameStore = create((set, get) => ({
  // Stato personaggio (dal backend)
  character: null,
  inventory: null,
  session: null,

  // Skill catalog + loadout
  skills: [],           // SkillView[] dal backend
  skillSlots: 3,        // massimo slot nel loadout

  // Chat UI
  messages: [],
  isStreaming: false,
  pendingText: '',

  // Notifiche UI
  uiEvents: [],
  levelUpInfo: null,    // { newLevel }
  lootInfo: null,       // { gold, items[] }
  overdrive: false,     // true quando tactical tension >= 80
  announcements: [],    // WS broadcast: [{ id, text, level }]
  specChoice: null,     // { tier, level, options[] } — pending spec choice

  // --- Azioni ---

  setGameState(character, inventory, session, specChoice = null) {
    set({ character, inventory, session, specChoice })
  },
  setSpecChoice: (choice) => set({ specChoice: choice }),

  clearGameState() {
    set({
      character: null, inventory: null, session: null,
      messages: [], isStreaming: false, pendingText: '',
      skills: [], lootInfo: null,
    })
  },

  setSkills(skills, skillSlots) {
    set({ skills, skillSlots: skillSlots ?? get().skillSlots })
  },

  // Aggiorna la lista locale delle skill (dopo PUT /loadout)
  updateLoadout(newLoadout) {
    set((state) => ({
      skills: state.skills.map((s) => ({
        ...s,
        is_in_loadout: newLoadout.includes(s.id),
      })),
      session: state.session ? { ...state.session, skill_loadout: newLoadout } : state.session,
    }))
  },

  startTurn(playerMessage) {
    set((state) => ({
      messages: [...state.messages, { role: 'player', text: playerMessage, id: Date.now() }],
      isStreaming: true,
      pendingText: '',
      uiEvents: [],
      levelUpInfo: null,
    }))
  },

  appendToken(text) {
    set((state) => ({ pendingText: state.pendingText + text }))
  },

  finishTurn(narrative, character, inventory, session, uiEvents, levelUp, newLevel, loot, overdrive) {
    // Dopo ogni turno ricalcola lo stato cooldown sulle skill
    const currentSkills = get().skills
    const updatedSkills = currentSkills.map((s) => {
      const cdRemaining = character?.skill_cooldowns?.[s.id] ?? 0
      return { ...s, cooldown_remaining: cdRemaining }
    })

    set((state) => ({
      messages: [...state.messages, { role: 'gm', text: narrative, id: Date.now() }],
      character,
      inventory,
      session,
      isStreaming: false,
      pendingText: '',
      uiEvents: uiEvents || [],
      levelUpInfo: levelUp ? { newLevel } : null,
      lootInfo: loot ?? null,
      overdrive: overdrive ?? false,
      skills: updatedSkills,
    }))
  },

  failTurn(errorText) {
    set((state) => ({
      messages: [...state.messages, { role: 'system', text: `⚠ ${errorText}`, id: Date.now() }],
      isStreaming: false,
      pendingText: '',
    }))
  },

  clearLevelUpInfo() { set({ levelUpInfo: null }) },
  clearLootInfo()    { set({ lootInfo: null }) },

  addAnnouncement(ann) {
    set((state) => ({ announcements: [...state.announcements, ann] }))
    // Auto-dismiss dopo 6s
    setTimeout(() => {
      set((state) => ({ announcements: state.announcements.filter((a) => a.id !== ann.id) }))
    }, 6000)
  },
  dismissAnnouncement(id) {
    set((state) => ({ announcements: state.announcements.filter((a) => a.id !== id) }))
  },

  // Aggiorna il personaggio in-store dopo allocazione stat (senza reload completo)
  updateCharacter(character) {
    set({ character })
  },

  // Aggiorna l'inventario in-store (es. dopo equip/unequip)
  setInventory: (inventory) => set({ inventory }),

  // Aggiorna la sessione in-store (es. dopo enter/exit dungeon)
  updateSession(session) {
    set({ session })
  },
}))
