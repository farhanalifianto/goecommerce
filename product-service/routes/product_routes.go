package routes

import (
	"product-service/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterProductRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	pc := &controller.ProductController{DB: db}

	api := app.Group("/api")
	p := api.Group("/products")

	p.Get("/", pc.List)
	p.Get("/:id", pc.Get)
	p.Post("/", authMiddleware, pc.Create)
	p.Put("/:id", authMiddleware, pc.Update)
	p.Delete("/:id", authMiddleware, pc.Delete)
	p.Post("/:id/decrement", pc.DecrementStock)
}
