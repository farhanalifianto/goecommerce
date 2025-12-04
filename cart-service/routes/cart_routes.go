package routes

import (
	"cart-service/controller"
	"cart-service/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterCartRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	cc := controller.NewCartController()

	api := app.Group("/api")
	cart := api.Group("/cart")

	// List carts for the user
	cart.Get("/", authMiddleware, cc.List)

	// Create new cart
	cart.Post("/", authMiddleware, cc.Create)

	// Get single cart
	cart.Get("/:id", authMiddleware, cc.Get)

	// Update cart (products / status)
	cart.Put("/:id", authMiddleware, cc.Update)

	// Delete cart (if allowed by service)
	cart.Delete("/:id", authMiddleware, cc.Delete)

	// Admin only: get all carts
	cart.Get("/all", authMiddleware, middleware.RoleRequired("admin"), cc.GetAll)
}
