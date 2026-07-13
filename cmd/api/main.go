// @title GraDiOl API
// @version 1.0
// @description Backend API for Graphic Diagram Online (GraDiOl)
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	"fmt"
	"log"

	"github.com/RenzIP/Graphic-Diagram-Online/app"
)

func main() {
	// Initialize the full application (config, DB, DI, routes)
	instance, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer instance.Close()

	// Start server
	addr := fmt.Sprintf(":%s", instance.Cfg.Port)
	log.Printf("🚀 GraDiOl API starting in http://localhost%s", addr)
	log.Printf("   env=%s log_level=%s", instance.Cfg.Env, instance.Cfg.LogLevel)
	if err := instance.App.Listen(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
