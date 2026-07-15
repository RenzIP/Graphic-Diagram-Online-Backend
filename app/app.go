// Package app provides the shared application setup for GraDiOl API.
// Used by both the local dev server (cmd/api/main.go) and the
// Google Cloud Function entry point (function.go).
package app

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/RenzIP/Graphic-Diagram-Online/config"
	"github.com/RenzIP/Graphic-Diagram-Online/db"
	"github.com/RenzIP/Graphic-Diagram-Online/handler"
	"github.com/RenzIP/Graphic-Diagram-Online/model"
	redissvc "github.com/RenzIP/Graphic-Diagram-Online/redis"
	"github.com/RenzIP/Graphic-Diagram-Online/repository"
	"github.com/RenzIP/Graphic-Diagram-Online/router"
	"github.com/RenzIP/Graphic-Diagram-Online/service"
	"github.com/RenzIP/Graphic-Diagram-Online/ws"
)

// Instance holds the initialized Fiber app and DB connection.
type Instance struct {
	App   *fiber.App
	DB    *gorm.DB
	Cfg   *config.Config
	Redis *goredis.Client // nil when Redis is unavailable
}

// New creates a fully wired Fiber application with all middleware,
// routes, and dependency injection configured.
func New() (*Instance, error) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load: %w", err)
	}

	// Connect to database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database connect: %w", err)
	}

	// AutoMigrate models
	if err := database.AutoMigrate(&model.DocumentVersion{}); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	log.Println("Connected to Supabase/PostgreSQL")

	// --- Redis connection (graceful degradation) ---
	var redisClient *goredis.Client
	var lockSvc *redissvc.LockService
	var presenceSvc *redissvc.PresenceService

	if cfg.RedisURL != "" {
		rc, err := redissvc.Connect(cfg.RedisURL)
		if err != nil {
			log.Printf("⚠ Redis unavailable (%v) — running without Redis", err)
		} else {
			redisClient = rc
			lockSvc = redissvc.NewLockService(rc)
			presenceSvc = redissvc.NewPresenceService(rc)
			log.Println("Connected to Redis")
		}
	}

	// --- WebSocket Hub ---
	var hub *ws.Hub
	if lockSvc != nil && presenceSvc != nil {
		hub = ws.NewHubWithRedis(lockSvc, presenceSvc)
	} else {
		hub = ws.NewHub()
	}

	// --- JWT Validator for WebSocket ---
	validator := &ws.JWTValidator{Secret: cfg.JWTSecret}

	// --- Repository layer ---
	userRepo := repository.NewUserRepo(database)
	wsRepo := repository.NewWorkspaceRepo(database)
	projRepo := repository.NewProjectRepo(database)
	docRepo := repository.NewDocumentRepo(database)

	// --- Service layer ---
	authSvc := service.NewAuthService(userRepo)
	wsSvc := service.NewWorkspaceService(wsRepo, userRepo)
	projSvc := service.NewProjectService(projRepo, wsSvc)
	docSvc := service.NewDocumentService(docRepo, projRepo, wsSvc)

	// --- Handler layer ---
	handlers := router.Handlers{
		Health:    handler.NewHealthHandler(),
		Auth:      handler.NewAuthHandler(authSvc, cfg),
		Workspace: handler.NewWorkspaceHandler(wsSvc),
		Project:   handler.NewProjectHandler(projSvc),
		Document:  handler.NewDocumentHandler(docSvc),
		Admin:     handler.NewAdminHandler(database),
		Hub:       hub,
		Validator: validator,
		WSAuthz:   docSvc,
	}

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "GraDiOl API",
		ErrorHandler: fiberErrorHandler,
	})

	// Register routes with middleware stack
	router.Setup(app, cfg, handlers)

	return &Instance{
		App:   app,
		DB:    database,
		Cfg:   cfg,
		Redis: redisClient,
	}, nil
}

// Close gracefully shuts down the application (closes DB, Redis, etc).
func (inst *Instance) Close() {
	if inst.Redis != nil {
		if err := inst.Redis.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}
	if inst.DB != nil {
		db.Disconnect(inst.DB)
	}
}

// fiberErrorHandler returns JSON errors for any unhandled Fiber errors.
func fiberErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"code":    "INTERNAL_ERROR",
		"message": message,
	})
}

