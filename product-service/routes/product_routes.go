package routes

import (
	"product-service/controller"
	"product-service/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterProductRoutes(app *fiber.App,db *gorm.DB, authMiddleware fiber.Handler) {
	pc := controller.NewProductController()

	api := app.Group("/api")
	p := api.Group("/products")


	//products
	p.Get("/", pc.ListProducts)
	p.Post("/", authMiddleware,middleware.RoleRequired("admin"), pc.CreateProduct)
	p.Get("/:id", pc.GetProduct)
	p.Put("/:id", authMiddleware,middleware.RoleRequired("admin"), pc.UpdateProduct)
	p.Delete("/:id",authMiddleware,middleware.RoleRequired("admin"), pc.DeleteProduct)

	//categories
	category := p.Group("/category")
	category.Post("/", authMiddleware,middleware.RoleRequired("admin"), pc.CreateCategory)
	category.Put("/:id", pc.UpdateCategory)
	category.Delete("/:id", pc.DeleteCategory)
	category.Get("/", pc.ListCategories)

	//stock
	stock := p.Group("/stock")
	stock.Get("/:product_id", authMiddleware,middleware.RoleRequired("admin"), pc.GetStock)
	stock.Put("/", authMiddleware, pc.UpdateStock)
}
