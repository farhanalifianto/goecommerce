package controller

import (
	"auth-service/proto/auth"
	"context"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

type AuthController struct {
	Client auth.AuthServiceClient
}

func NewAuthController() *AuthController {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect to auth gRPC: " + err.Error())
	}
	client := auth.NewAuthServiceClient(conn)
	return &AuthController{Client: client}
}

func (ac *AuthController) Register(c *fiber.Ctx) error {
	var req auth.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	res, err := ac.Client.Register(context.Background(), &req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}

func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req auth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	res, err := ac.Client.Login(context.Background(), &req)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}

func (ac *AuthController) ValidateToken(c *fiber.Ctx) error {
	type TokenReq struct {
		Token string `json:"token"`
	}

	var req TokenReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	res, err := ac.Client.ValidateToken(context.Background(), &auth.ValidateTokenRequest{Token: req.Token})
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}
