// Command ws is a WebSocket-only entrypoint for GraDiOl realtime collaboration.
//
// It wires just enough of the stack to serve the /ws/:documentId endpoint:
// config, database (needed for workspace-membership authorization), optional
// Redis (locks + presence), the WS hub, and the JWT validator. It deliberately
// does NOT register the REST API, OAuth, admin, or Swagger routes — those stay
// on the existing deployment. This keeps the Cloud Run image focused on the one
// job Cloud Functions can't do: long-lived WebSocket connections.
//
// IMPORTANT: the hub broadcasts in-memory per instance. Run this with exactly
// one instance (Cloud Run: --min-instances=1 --max-instances=1). Scaling beyond
// one instance requires a Redis pub/sub relay, which is not implemented here.
package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/RenzIP/Graphic-Diagram-Online/config"
	"github.com/RenzIP/Graphic-Diagram-Online/db"
	redissvc "github.com/RenzIP/Graphic-Diagram-Online/redis"
	"github.com/RenzIP/Graphic-Diagram-Online/repository"
	"github.com/RenzIP/Graphic-Diagram-Online/service"
	"github.com/RenzIP/Graphic-Diagram-Online/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	// Database — required because WS authorization resolves
	// document → workspace → membership before admitting a client.
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}
	log.Println("Connected to Supabase/PostgreSQL")

	// Redis — optional. When present it backs node locks and presence;
	// when absent the hub falls back to in-memory state.
	var lockSvc *redissvc.LockService
	var presenceSvc *redissvc.PresenceService
	if cfg.RedisURL != "" {
		rc, err := redissvc.Connect(cfg.RedisURL)
		if err != nil {
			log.Printf("⚠ Redis unavailable (%v) — running without Redis", err)
		} else {
			defer func() { _ = rc.Close() }()
			lockSvc = redissvc.NewLockService(rc)
			presenceSvc = redissvc.NewPresenceService(rc)
			log.Println("Connected to Redis")
		}
	}

	// WS hub (Redis-backed when available).
	var hub *ws.Hub
	if lockSvc != nil && presenceSvc != nil {
		hub = ws.NewHubWithRedis(lockSvc, presenceSvc)
	} else {
		hub = ws.NewHub()
	}

	validator := &ws.JWTValidator{Secret: cfg.JWTSecret}

	// Minimal DI needed only for the membership authorizer.
	wsRepo := repository.NewWorkspaceRepo(database)
	userRepo := repository.NewUserRepo(database)
	projRepo := repository.NewProjectRepo(database)
	docRepo := repository.NewDocumentRepo(database)
	wsSvc := service.NewWorkspaceService(wsRepo, userRepo)
	docSvc := service.NewDocumentService(docRepo, projRepo, wsSvc)

	app := fiber.New(fiber.Config{
		AppName:               "GraDiOl WS",
		DisableStartupMessage: true,
	})

	// Health check for Cloud Run readiness/liveness probes.
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// The one route this service exists for.
	app.Use("/ws", ws.UpgradeMiddleware())
	app.Get("/ws/:documentId", ws.HandleWebSocket(hub, validator, docSvc))

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🔌 GraDiOl WS listening on %s (env=%s)", addr, cfg.Env)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("failed to start WS server: %v", err)
	}
}
