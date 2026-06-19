import ChatPanel from '../panels/ChatPanel.jsx'
import StatsPanel from '../panels/StatsPanel.jsx'
import InventoryPanel from '../panels/InventoryPanel.jsx'
import SkillPanel from '../panels/SkillPanel.jsx'
import QuestPanel from '../panels/QuestPanel.jsx'
import DungeonPanel from '../panels/DungeonPanel.jsx'
import WorldPanel from '../panels/WorldPanel.jsx'
import MarketPanel from '../panels/MarketPanel.jsx'
import SkillTreePanel from '../panels/SkillTreePanel.jsx'

export const PANELS = {
  chat:      { id: 'chat',      label: 'Chat',        icon: '💬', component: ChatPanel,      minW: 4, minH: 8 },
  stats:     { id: 'stats',     label: 'Stats',       icon: '📊', component: StatsPanel,     minW: 2, minH: 5 },
  inventory: { id: 'inventory', label: 'Inventario',  icon: '🎒', component: InventoryPanel, minW: 2, minH: 4 },
  skill:     { id: 'skill',     label: 'Skill',       icon: '⚡', component: SkillPanel,     minW: 2, minH: 4 },
  quest:     { id: 'quest',     label: 'Quest',       icon: '📜', component: QuestPanel,     minW: 2, minH: 4 },
  dungeon:   { id: 'dungeon',   label: 'Dungeon',     icon: '🗺', component: DungeonPanel,   minW: 3, minH: 5 },
  world:     { id: 'world',     label: 'Mondo',       icon: '🌍', component: WorldPanel,     minW: 2, minH: 4 },
  market:    { id: 'market',    label: 'Mercato',     icon: '🏪', component: MarketPanel,    minW: 2, minH: 6 },
  skilltree: { id: 'skilltree', label: 'Albero Skill', icon: '🌳', component: SkillTreePanel, minW: 3, minH: 8 },
}

// rowHeight=50, cols=12. Numeri pensati per 1080p (area gioco ~900px tall).
export const PRESETS = {
  default: {
    lg: [
      // Riga alta: Stats | Chat | Inventario + Skill
      { i: 'stats',     x: 0, y: 0,  w: 3, h: 8,  minW: 2, minH: 5 },
      { i: 'chat',      x: 3, y: 0,  w: 6, h: 8,  minW: 4, minH: 8 },
      { i: 'inventory', x: 9, y: 0,  w: 3, h: 4,  minW: 2, minH: 4 },
      { i: 'skill',     x: 9, y: 4,  w: 3, h: 4,  minW: 2, minH: 4 },
      // Riga media: Quest | Mondo
      { i: 'quest',     x: 0, y: 8,  w: 6, h: 4,  minW: 2, minH: 4 },
      { i: 'world',     x: 6, y: 8,  w: 6, h: 4,  minW: 2, minH: 4 },
      // Riga bassa: Dungeon | Mercato | Albero Skill
      { i: 'dungeon',   x: 0, y: 12, w: 4, h: 8,  minW: 3, minH: 5 },
      { i: 'market',    x: 4, y: 12, w: 4, h: 8,  minW: 2, minH: 6 },
      { i: 'skilltree', x: 8, y: 12, w: 4, h: 8,  minW: 3, minH: 8 },
    ],
  },
  combat: {
    lg: [
      { i: 'chat',      x: 0, y: 4, w: 8, h: 9,  minW: 4, minH: 6 },
      { i: 'stats',     x: 0, y: 0, w: 5, h: 4,  minW: 2, minH: 3 },
      { i: 'skill',     x: 5, y: 0, w: 3, h: 4,  minW: 2, minH: 3 },
      { i: 'inventory', x: 8, y: 0, w: 4, h: 6,  minW: 2, minH: 4 },
      { i: 'quest',     x: 8, y: 6, w: 4, h: 7,  minW: 2, minH: 4 },
    ],
  },
  exploration: {
    lg: [
      { i: 'chat',      x: 0, y: 0, w: 7, h: 13, minW: 4, minH: 8 },
      { i: 'quest',     x: 7, y: 0, w: 5, h: 6,  minW: 2, minH: 4 },
      { i: 'stats',     x: 7, y: 6, w: 5, h: 3,  minW: 2, minH: 3 },
      { i: 'inventory', x: 7, y: 9, w: 3, h: 4,  minW: 2, minH: 4 },
      { i: 'skill',     x: 10, y: 9, w: 2, h: 4, minW: 2, minH: 4 },
    ],
  },
  dungeon: {
    lg: [
      { i: 'chat',    x: 0, y: 0, w: 6, h: 13, minW: 4, minH: 8 },
      { i: 'dungeon', x: 6, y: 0, w: 6, h: 7,  minW: 3, minH: 5 },
      { i: 'stats',   x: 6, y: 7, w: 3, h: 6,  minW: 2, minH: 5 },
      { i: 'skill',   x: 9, y: 7, w: 3, h: 6,  minW: 2, minH: 4 },
    ],
  },
}
