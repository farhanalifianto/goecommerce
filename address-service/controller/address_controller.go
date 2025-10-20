package controller

import (
	pb "address-service/proto/address"
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

type AddressController struct {
	Client pb.AddressServiceClient
}

func (ac *AddressController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.ListAddresses(ctx, &pb.ListAddressRequest{
		OwnerId: uint32(userID),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Addresses)
}

func (ac *AddressController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.GetAddress(ctx, &pb.GetAddressRequest{Id: uint32(id)})
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Address)
}

func (ac *AddressController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var body struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.CreateAddress(ctx, &pb.CreateAddressRequest{
		Name:     body.Name,
		Desc:     body.Desc,
		OwnerId:  uint32(userID),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(resp.Address)
}

func (ac *AddressController) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.UpdateAddress(ctx, &pb.UpdateAddressRequest{
		Id:   uint32(id),
		Name: body.Name,
		Desc: body.Desc,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Address)
}

func (ac *AddressController) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.DeleteAddress(ctx, &pb.DeleteAddressRequest{Id: uint32(id)})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}

func NewAddressController() *AddressController {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect to address gRPC: " + err.Error())
	}

	client := pb.NewAddressServiceClient(conn)
	return &AddressController{Client: client}
}
