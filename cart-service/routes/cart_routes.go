package routes

import (
	"cart-service/controller"

	"github.com/gofiber/fiber/v2"
)

func RegisterCartRoutes(app *fiber.App, auth fiber.Handler) {
	cc := controller.NewCartController()
	api := app.Group("/api/cart")

	api.Post("/", auth, cc.Create)
	api.Get("/", auth, cc.GetCart)
	api.Delete("/:id", auth, cc.DeleteCart)
}
