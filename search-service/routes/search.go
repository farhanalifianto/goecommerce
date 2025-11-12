package routes

import (
	"context"
	"log"
	"strings"

	"encoding/json"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
)

func RegisterSearchRoutes(app *fiber.App, es *elasticsearch.Client) {
	app.Get("/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return c.Status(400).JSON(fiber.Map{"error": "query parameter 'q' is required"})
		}

		// 🔍 Build Elasticsearch search body
		body := `{
			"query": {
				"multi_match": {
					"query": "` + query + `",
					"fields": ["name", "desc"]
				}
			}
		}`

		res, err := es.Search(
			es.Search.WithContext(context.Background()),
			es.Search.WithIndex("addresses"),
			es.Search.WithBody(strings.NewReader(body)),
			es.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			log.Printf("❌ Elasticsearch search error: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "search failed"})
		}
		defer res.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to parse response"})
		}

		return c.JSON(result)
	})
}
