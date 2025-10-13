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

	w.Post("/", authMiddleware,wc.CreateOrUpdate)
	w.Get("/", authMiddleware,wc.Get)
	w.Delete("/:id", authMiddleware,wc.DeleteProduct)

}
