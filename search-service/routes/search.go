package routes

import (
	"search-service/elasticsearch"
	"search-service/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterSearchRoutes(app *fiber.App, authMiddleware fiber.Handler, esClient *elasticsearch.ElasticClient) {
	api := app.Group("/api")
	s := api.Group("/search")

	//admin only
	s.Get("/address", authMiddleware, middleware.RoleRequired("admin"), func(c *fiber.Ctx) error {
		q := c.Query("q")
		if q == "" {
			return c.Status(400).JSON(fiber.Map{"error": "missing query parameter ?q="})
		}

		results, err := esClient.SearchAddresses(q)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(results)
	})
}