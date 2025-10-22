package controller

import (
	"context"
	pb "user-service/proto/user"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

type UserController struct {
	Client pb.UserServiceClient
}



func (uc *UserController) Register(c *fiber.Ctx) error {
	var req pb.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	res, err := uc.Client.Register(context.Background(), &req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (uc *UserController) Login(c *fiber.Ctx) error {
	var req pb.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	res, err := uc.Client.Login(context.Background(), &req)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (uc *UserController) Me(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	res, err := uc.Client.GetMe(context.Background(), &pb.GetMeRequest{Id: uint32(userID)})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res)
}

func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	res, err := uc.Client.GetUsers(context.Background(), &pb.Empty{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(res.Users)
}

func NewUserController() *UserController {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure()) // port gRPC user
	if err != nil {
		panic("failed to connect to user gRPC: " + err.Error())
	}
	client := pb.NewUserServiceClient(conn)
	return &UserController{Client: client}
}