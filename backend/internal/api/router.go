package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"daydream/internal/agents"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/game"
	"daydream/internal/queue"
	"daydream/internal/ws"
)

// Handler raccoglie le dipendenze condivise tra tutti gli handler.
type Handler struct {
	DB         db.DBClient
	Queue      *queue.PlayerQueue
	Skills     *game.SkillRegistry
	Hub        *ws.Hub
	DungeonGen *agents.DungeonGenerator // nil se non configurato
}

// WithDungeonGenerator è un'opzione funzionale per configurare il DungeonGenerator.
func WithDungeonGenerator(dg *agents.DungeonGenerator) func(*Handler) {
	return func(h *Handler) { h.DungeonGen = dg }
}

func NewRouter(dbClient db.DBClient, playerQueue *queue.PlayerQueue, skills *game.SkillRegistry, hub *ws.Hub, opts ...func(*Handler)) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	h := &Handler{DB: dbClient, Queue: playerQueue, Skills: skills, Hub: hub}
	for _, opt := range opts {
		opt(h)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "ws_clients": hub.ConnectedCount()})
	})

	// WebSocket — autenticazione via query param ?token=<jwt>
	r.GET("/ws", hub.ServeWS)

	// Auth (pubblici)
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
	}

	// Endpoint protetti da JWT
	api := r.Group("/api")
	api.Use(auth.Middleware())
	{
		// Auth
		api.GET("/auth/me", h.Me)

		// Personaggio
		api.POST("/character", h.CreateCharacter)
		api.GET("/character", h.GetCharacter)
		api.PUT("/character/loadout", h.UpdateLoadout)
		api.PUT("/character/stats", h.AllocateStats)
		api.GET("/character/spec-choices", h.GetSpecChoices)
		api.POST("/character/spec-choice", h.ChooseSpec)
		api.GET("/character/skill-tree", h.GetSkillTree)
		api.POST("/character/skill-tree/unlock", h.UnlockSkillTreeNode)
		api.POST("/character/custom-skill/upgrade", h.UpgradeCustomSkill)

		// Skill
		api.GET("/skills", h.GetSkills)

		// Quest
		api.GET("/quests", h.GetQuests)

		// Inventario
		api.POST("/inventory/equip", h.EquipItem)
		api.POST("/inventory/unequip", h.UnequipItem)
		api.POST("/inventory/appraise", h.AppraiseItem)

		// Gioco
		api.POST("/chat", ChatRateLimit(), h.Chat)
		api.GET("/state", h.GetState)

		// Dungeon
		api.GET("/dungeon", h.ListDungeons)
		api.POST("/dungeon/enter", h.EnterDungeon)
		api.GET("/dungeon/map", h.DungeonMap)
		api.POST("/dungeon/exit", h.ExitDungeon)

		// Catalogo meccanico
		api.GET("/catalog/classes", h.ListClasses)
		api.GET("/catalog/monsters", h.ListMonsters)
		api.GET("/catalog/diary", h.ListDiary)
		api.GET("/catalog/bestiary", h.ListBestiary)
		api.GET("/catalog/npcs", h.ListNPCs)

		// World flags
		api.GET("/world/flags", h.GetWorldFlags)

		// Mercato
		api.GET("/market/browse", h.BrowseMarket)
		api.POST("/market/buy", h.BuyMarketItem)
		api.POST("/market/negotiate", h.NegotiateMarketItem)
		api.POST("/market/analyze", h.AnalyzeMarketItem)
	}

	return r
}
