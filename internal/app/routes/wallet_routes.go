package routes

import (
	"grls/internal/modules/wallet/handler"

	"github.com/gofiber/fiber/v2"
)

func NewWalletRoutes(routerPoint fiber.Router, handler *handler.WalletHandler) {
	routerPoint.Post("/deposit", handler.Deposit)
}
