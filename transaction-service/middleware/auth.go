package middleware

import (
	"context"
	"strings"
	authpb "transaction-service/proto/auth"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

func AuthMiddleware() fiber.Handler {
	conn, err := grpc.Dial("auth-service:50052", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect to auth-service gRPC: " + err.Error())
	}
	client := authpb.NewAuthServiceClient(conn)

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		res, err := client.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{
			Token: token,
		})
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("user_id", res.Id)
		c.Locals("email", res.Email)
		c.Locals("role", res.Role)

		return c.Next()
	}
}

func RoleRequired(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role")
		if userRole != role {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden: admin only"})
		}
		return c.Next()
	}
}