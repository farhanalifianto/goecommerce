package routes

import (
	"address-service/controller"
	"address-service/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterAddressRoutes(app *fiber.App, db *gorm.DB, authMiddleware fiber.Handler) {
	ac := controller.NewAddressController()

	api := app.Group("/api")
	a := api.Group("/address")

	a.Get("/", authMiddleware, ac.List)
	a.Post("/", authMiddleware, ac.Create)
	a.Get("/all", authMiddleware,middleware.RoleRequired("admin"), ac.GetAllAddresses)
	// a.Get("/all", authMiddleware,middleware.RoleRequired("admin"), ac.GetAllAddress)
	a.Get("/:id", authMiddleware, ac.Get)
	a.Put("/:id", authMiddleware, ac.Update)
	a.Delete("/:id", authMiddleware, ac.Delete)
}
