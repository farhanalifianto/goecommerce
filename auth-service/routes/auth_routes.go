package routes

import (
	"auth-service/controller"

	"github.com/gofiber/fiber/v2"
)

func RegisterAuthRoutes(app *fiber.App) {
	ac := controller.NewAuthController()

	api := app.Group("/api")
	a := api.Group("/auth")

	// Semua endpoint autentikasi ada di sini
	a.Post("/register", ac.Register)
	a.Post("/login", ac.Login)
	a.Post("/validate", ac.ValidateToken)
}
