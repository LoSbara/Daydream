package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"daydream/internal/agents"
	"daydream/internal/api"
	"daydream/internal/config"
	"daydream/internal/db"
	"daydream/internal/embedding"
	"daydream/internal/game"
	"daydream/internal/llm"
	"daydream/internal/queue"
	"daydream/internal/rag"
	"daydream/internal/ws"
)

func main() {
	// Carica .env se presente (sviluppo locale)
	if err := godotenv.Load(); err != nil {
		log.Println("nessun .env trovato, uso variabili d'ambiente di sistema")
	}

	// Structured logging: JSON in produzione, testo leggibile in dev
	setupLogger()

	slog.Info("avvio", "app", config.AppName)

	// Connessione SurrealDB con retry (necessario se SurrealDB sta ancora avviandosi)
	dbClient, err := connectWithRetry(10, 2*time.Second)
	if err != nil {
		slog.Error("impossibile connettersi a SurrealDB", "err", err)
		os.Exit(1)
	}
	slog.Info("SurrealDB connesso")

	// Migrazione schema (idempotente)
	if err := dbClient.Migrate(); err != nil {
		slog.Error("migrazione schema fallita", "err", err)
		os.Exit(1)
	}
	slog.Info("schema applicato")

	// LLM provider
	llmProvider := buildLLMProvider()
	slog.Info("LLM provider pronto", "provider", llmProvider.Name(), "model", os.Getenv("LLM_MODEL"))

	// Skill registry
	skills, err := game.NewSkillRegistry()
	if err != nil {
		slog.Error("impossibile caricare skill registry", "err", err)
		os.Exit(1)
	}
	slog.Info("skill registry caricato", "count", len(skills.All()))

	// Embedding provider + RAG (disabilitato di default)
	var retriever *rag.Retriever
	var contentGen *agents.ContentGenerator
	if os.Getenv("EMBED_ENABLED") == "true" {
		embedDims, _ := strconv.Atoi(os.Getenv("EMBED_DIMENSIONS"))
		embedProvider := embedding.NewOpenAICompat(
			os.Getenv("EMBED_BASE_URL"),
			os.Getenv("EMBED_API_KEY"),
			os.Getenv("EMBED_MODEL"),
			embedDims,
		)
		slog.Info("embedding provider attivo", "provider", embedProvider.Name(), "dims", embedProvider.Dimensions())

		retriever = rag.NewRetriever(dbClient, embedProvider)
		seeder := rag.NewSeeder(dbClient, embedProvider)
		if err := seeder.Seed(context.Background()); err != nil {
			slog.Warn("knowledge_base seeding fallito, RAG degradato", "err", err)
			retriever = nil
		} else {
			contentGen = agents.NewContentGenerator(dbClient, llmProvider, embedProvider, retriever)
			slog.Info("content generator attivo")
		}
	} else {
		slog.Info("RAG disabilitato (EMBED_ENABLED=true per abilitarlo)")
	}

	// Dungeon Generator — richiede solo il provider LLM, sempre attivo
	dungeonGen := agents.NewDungeonGenerator(llmProvider)
	slog.Info("dungeon generator attivo")

	// WebSocket hub
	hub := ws.NewHub()
	go hub.Run()
	slog.Info("WebSocket hub avviato")

	// Agent system
	validator := agents.NewValidator(dbClient)
	compactor := agents.NewCompactor(dbClient, llmProvider)

	// Game engine
	engine := game.NewEngine(dbClient, llmProvider, skills, retriever).
		WithAgents(validator, compactor).
		WithContentGenerator(contentGen)

	// Player FIFO queue
	playerQueue := queue.New(engine.AsTurnProcessor())

	// Router Gin
	router := api.NewRouter(dbClient, playerQueue, skills, hub,
		api.WithDungeonGenerator(dungeonGen),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server in ascolto", "app", config.AppName, "port", port)
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// setupLogger configura slog globalmente.
// LOG_FORMAT=json → JSON strutturato (prod); altrimenti testo leggibile (dev).
func setupLogger() {
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	slog.SetDefault(slog.New(handler))
}

// buildLLMProvider seleziona il provider in base a LLM_PROVIDER.
// Supporta: openai-compat (default), anthropic.
func buildLLMProvider() llm.Provider {
	switch os.Getenv("LLM_PROVIDER") {
	case "anthropic":
		model := os.Getenv("LLM_MODEL")
		if model == "" {
			model = "claude-sonnet-4-6"
		}
		return llm.NewAnthropic(os.Getenv("LLM_API_KEY"), model)
	default:
		return llm.NewOpenAI(
			os.Getenv("LLM_BASE_URL"),
			os.Getenv("LLM_API_KEY"),
			os.Getenv("LLM_MODEL"),
		)
	}
}

func connectWithRetry(attempts int, delay time.Duration) (*db.Client, error) {
	for i := 1; i <= attempts; i++ {
		client, err := db.New()
		if err == nil {
			return client, nil
		}
		slog.Warn("SurrealDB non disponibile, riprovo", "attempt", i, "max", attempts, "err", err)
		if i < attempts {
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("SurrealDB non raggiungibile dopo %d tentativi", attempts)
}
