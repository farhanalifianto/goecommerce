package controller

import (
	"context"
	"fmt"
	"log"

	pb "cart-service/proto/cart"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

type CartController struct {
	Client pb.CartServiceClient
}

func NewCartController() *CartController {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}
	client := pb.NewCartServiceClient(conn)
	return &CartController{Client: client}
}

func (cc *CartController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var body struct {
		Products []struct {
			ProductID uint `json:"product_id"`
			Qty       uint `json:"qty"`
		} `json:"products"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	req := &pb.CreateCartRequest{
		OwnerId: uint32(userID),
	}

	for _, p := range body.Products {
		req.Products = append(req.Products, &pb.Product{
			ProductId: uint32(p.ProductID),
			Qty:       uint32(p.Qty),
		})
	}

	resp, err := cc.Client.CreateCart(context.Background(), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Cart)
}

func (cc *CartController) GetCart(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	resp, err := cc.Client.GetCart(context.Background(), &pb.GetCartRequest{OwnerId: uint32(userID)})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(resp.Cart)
}

func (cc *CartController) DeleteCart(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var productID uint
	if _, err := fmt.Sscanf(c.Params("id"), "%d", &productID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid product id"})
	}

	resp, err := cc.Client.DeleteProduct(context.Background(), &pb.DeleteProductRequest{
		OwnerId:   uint32(userID),
		ProductId: uint32(productID),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Cart)
}
