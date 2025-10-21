package routes

import (
	"product-service/controller"
	"product-service/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterProductRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	pc := &controller.ProductController{DB: db}

	api := app.Group("/api")
	p := api.Group("/products")

	p.Get("/", pc.List)
	p.Get("/:id", pc.Get)
	p.Post("/", authMiddleware, middleware.RoleRequired("admin"), pc.Create)
	p.Put("/:id", authMiddleware,middleware.RoleRequired("admin"), pc.Update)
	p.Delete("/:id", authMiddleware,middleware.RoleRequired("admin"), pc.Delete)
	// p.Post("/:id/reduce", pc.ReduceStock)
}
