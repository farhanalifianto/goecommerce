package routes

import (
	"user-service/controller"

	"github.com/gofiber/fiber/v2"
)

func RegisterUserRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	uc := controller.NewUserController()

	api := app.Group("/api")
	u := api.Group("/users")

	u.Post("/register", uc.Register)
	u.Post("/login", uc.Login)
	u.Get("/me", authMiddleware, uc.Me)
	u.Get("/", authMiddleware, uc.GetUsers)
}
