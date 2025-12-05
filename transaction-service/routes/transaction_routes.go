package routes

import (
	"transaction-service/controller"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
)

func RegisterTransactionRoutes(app *fiber.App,db *gorm.DB, authMiddleware fiber.Handler) {
	tc := controller.NewTransactionController()

	api := app.Group("/api")
	t := api.Group("/transaction")
	t.Post("/", authMiddleware, tc.Create)
	
	t.Get("/", authMiddleware, tc.ListUser)
	t.Get("/all", authMiddleware, tc.ListAll)
	t.Post("/:id/cancel", authMiddleware, tc.Cancel)
	t.Get("/:id", authMiddleware, tc.Get)
	// t.Post("/:id/pay", authMiddleware, tc.Pay)
}
