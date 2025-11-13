package routes

import (
	"search-service/elasticsearch"

	"github.com/gofiber/fiber/v2"
)

func RegisterSearchRoutes(app *fiber.App, esClient *elasticsearch.ElasticClient) {
	app.Get("/api/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return c.Status(400).JSON(fiber.Map{"error": "missing query parameter ?q="})
		}

		results, err := esClient.SearchAddresses(query)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(results)
	})
}
