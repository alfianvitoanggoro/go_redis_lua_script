package app

import (
	"grls/internal/app/factory"
	"grls/internal/app/routes"
	"grls/internal/config"
	"grls/pkg/logger"
	"net"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber  *fiber.App
	Config *config.Config
}

func NewApp(cfg *config.Config, dbWrite *gorm.DB, dbRead *gorm.DB, rdb *redis.UniversalClient) *App {
	// Build the application factory
	container := factory.Build(dbWrite, dbRead, *rdb)

	// Create a new Fiber app
	fiberApp := fiber.New()

	app := &App{Fiber: fiberApp, Config: cfg}

	// Register routes
	routes.NewRoutes(fiberApp, container)

	return app
}

func (a *App) Start(listener net.Listener) {
	configApp := a.Config.App

	logger.Infof("✅ %s server started on port: %s", configApp.Name, configApp.Port)

	// Jalankan Fiber
	if err := a.Fiber.Listener(listener); err != nil {
		errDetail := err.Error()
		logger.WriteLogToFile("failed", "app.Start", map[string]any{
			"app_name": configApp.Name,
			"app_port": configApp.Port,
		}, &errDetail)
		logger.Fatal("❌ Failed to start server: " + err.Error())
	}
}
