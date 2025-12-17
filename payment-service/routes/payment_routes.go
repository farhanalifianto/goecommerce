package routes

import (
	"payment-service/controller"
	"payment-service/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterPaymentRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	pc := controller.NewPaymentController()

	api := app.Group("/api")
	p := api.Group("/payment")

	// =========================
	// USER ROUTES
	// =========================
	p.Get("/", authMiddleware, pc.List)        // list payment user
	p.Post("/", authMiddleware, pc.Create)     // create payment (by transaction_id)
	p.Post("/:id/pay", authMiddleware, pc.Pay) // bayar payment (manual)

	// =========================
	// ADMIN ROUTE
	// =========================
	p.Get(
		"/all",
		authMiddleware,
		middleware.RoleRequired("admin"),
		pc.ListAll,
	)

	// =========================
	// SINGLE RESOURCE
	// =========================
	p.Get("/:id", authMiddleware, pc.Get)
}
