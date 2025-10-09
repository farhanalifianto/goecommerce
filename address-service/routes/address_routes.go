package routes

import (
	"address-service/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterAddressRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	ad := &controller.AddressController{DB: db}

	api := app.Group("/api")
	a := api.Group("/address")

	a.Get("/", authMiddleware,ad.List)
	a.Post("/", authMiddleware, ad.Create)
	a.Get("/:id", authMiddleware,ad.Get)
	a.Put("/:id", authMiddleware, ad.Update)
	a.Delete("/:id", authMiddleware, ad.Delete)
}
