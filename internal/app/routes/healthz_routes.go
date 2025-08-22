package routes

import (
	"grls/pkg/response"

	"github.com/gofiber/fiber/v2"
)

func NewHealthzRoutes(routerHealthz fiber.Router) {
	routerHealthz.Get("/", func(c *fiber.Ctx) error {
		return response.WriteSuccess(c, fiber.StatusOK, "API is healthy", nil)
	})
}
