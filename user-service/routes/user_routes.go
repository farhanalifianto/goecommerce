package routes

import (
	"user-service/controller"
	"user-service/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterUserRoutes(app *fiber.App, db *gorm.DB, jwtSecret string) {
	uc := &controller.UserController{DB: db, JWTSecret: jwtSecret}

	api := app.Group("/api")
	u := api.Group("/users")

	u.Post("/register", uc.Register)
	u.Post("/login", uc.Login)
	u.Get("/me", middleware.AuthRequired(jwtSecret), uc.Me)
}
