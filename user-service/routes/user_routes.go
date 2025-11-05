package routes

import (
	"user-service/controller"
	"user-service/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterUserRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	uc := controller.NewUserController()

	api := app.Group("/api")
	u := api.Group("/users")

	// Hanya user terautentikasi yang bisa akses /me
	u.Get("/me", authMiddleware, uc.Me)

	// Hanya admin yang bisa akses semua user
	u.Get("/all", authMiddleware, middleware.RoleRequired("admin"), uc.GetUsers)
}
