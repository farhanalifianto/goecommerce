package routes

import (
	"transaction-service/controller"
	"transaction-service/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupTransactionRoutes(app *fiber.App, tc *controller.TransactionController, userServiceURL string) {
	api := app.Group("/api/transactions", middleware.AuthRequired(userServiceURL))

// 	api.Post("/", tc.CreateTransaction)
// 	api.Get("/", tc.GetUserTransactions)
// 	api.Get("/:id", tc.GetTransactionByID)
// 
}
