package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

var userServiceURL = getEnv("USER_SERVICE_URL", "http://user-service:3001")

func AuthRequired(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return c.Status(401).JSON(fiber.Map{"error": "missing auth"})
	}

	req, _ := http.NewRequest("GET", userServiceURL+"/api/users/me", nil)
	req.Header.Set("Authorization", auth)
	client := &http.Client{Timeout: time.Second * 5}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "auth failed"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	var user struct {
		ID    uint   `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(buf.Bytes(), &user); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "decode failed"})
	}

	c.Locals("user_id", user.ID)
	c.Locals("user_role", user.Role)
	return c.Next()
}

func RoleRequired(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole == nil {
			return c.Status(403).JSON(fiber.Map{"error": "no role"})
		}

		role := userRole.(string)
		for _, r := range roles {
			if role == r {
				return c.Next()
			}
		}
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
