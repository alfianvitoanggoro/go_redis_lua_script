package routes

import (
	"grls/internal/app/factory"

	"github.com/gofiber/fiber/v2"
)

func NewRoutes(app *fiber.App, container *factory.Container) {
	routerApi := app.Group("/api")

	// Register healthz routes
	healthzRoutes := routerApi.Group("/healthz")
	NewHealthzRoutes(healthzRoutes)

	// Wallet Routes
	routerWallet := routerApi.Group("/wallet")
	NewWalletRoutes(routerWallet, container.WalletHandler)

}
