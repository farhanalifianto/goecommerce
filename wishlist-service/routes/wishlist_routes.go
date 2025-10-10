package routes

import (
	"wishlist-service/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterWishlistRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	wc := &controller.WishlistController{DB: db}

	api := app.Group("/api")
	w := api.Group("/wishlist")

	w.Get("/", authMiddleware,wc.List)
	w.Post("/", authMiddleware, wc.Create)
	w.Get("/:id", authMiddleware,wc.Get)
	w.Put("/:id", authMiddleware, wc.Update)
	w.Delete("/:id", authMiddleware, wc.Delete)
}
