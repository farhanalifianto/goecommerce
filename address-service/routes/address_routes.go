package routes

import (
	"address-service/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterAddressRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	ac := controller.NewAddressController() // sekarang lewat gRPC client

	api := app.Group("/api")
	a := api.Group("/address")

	a.Get("/", authMiddleware, ac.List)
	a.Post("/", authMiddleware, ac.Create)
	a.Get("/:id", authMiddleware, ac.Get)
	a.Put("/:id", authMiddleware, ac.Update)
	a.Delete("/:id", authMiddleware, ac.Delete)
}
