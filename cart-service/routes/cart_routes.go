package routes

import (
	"cart-service/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterCartRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	cc := &controller.CartController{DB: db}

	api := app.Group("/api")
	c := api.Group("/cart")

	c.Post("/", authMiddleware, cc.Create)
	c.Get("/", authMiddleware, cc.GetCart)
	c.Delete("/:id", authMiddleware, cc.DeleteCart)
}
