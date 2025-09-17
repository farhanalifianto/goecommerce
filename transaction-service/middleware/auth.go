package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AuthRequired(userServiceURL string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing auth"})
		}

		req, _ := http.NewRequest("GET", userServiceURL+"/api/users/me", nil)
		req.Header.Set("Authorization", auth)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("auth call failed:", err)
			return c.Status(401).JSON(fiber.Map{"error": "auth failed"})
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		var user struct {
			ID   uint   `json:"id"`
			Role string `json:"role"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "decode failed"})
		}

		c.Locals("user_id", user.ID)
		c.Locals("user_role", user.Role)

		return c.Next()
	}
}
