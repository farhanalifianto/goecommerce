package routes

import (
	"cart-service/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterAddressRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	cc := &controller.CartController{DB: db}

	api := app.Group("/api")
	c := api.Group("/cart")

	c.Get("/", authMiddleware,cc.List)
	c.Post("/", authMiddleware, cc.Create)
	c.Get("/:id", authMiddleware,cc.Get)
	c.Put("/:id", authMiddleware, cc.Update)
	c.Delete("/:id", authMiddleware, cc.Delete)
}
