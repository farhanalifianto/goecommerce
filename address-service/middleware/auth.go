package middleware

import (
	"context"
	"strings"

	pb "address-service/proto/user"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

func AuthRequired(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(401).JSON(fiber.Map{"error": "missing or invalid token"})
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	conn, err := grpc.Dial("user-service:50051", grpc.WithInsecure())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to connect auth service"})
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	res, err := client.ValidateToken(context.Background(), &pb.ValidateTokenRequest{Token: token})
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}

	// simpan user info di context
	c.Locals("user_id", res.Id)
	c.Locals("user_email", res.Email)
	c.Locals("user_role", res.Role)

	return c.Next()
}
func RoleRequired(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("user_role")
		if role == nil {
			return c.Status(401).JSON(fiber.Map{"error": "missing role info"})
		}

		userRole := role.(string)
		for _, allowed := range roles {
			if userRole == allowed {
				return c.Next()
			}
		}

		return c.Status(403).JSON(fiber.Map{"error": "forbidden: insufficient role"})
	}
}
