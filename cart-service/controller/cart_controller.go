package controller

import (
	"cart-service/grpc_client"
	pb "cart-service/proto/cart"
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CartController struct {
	Client     pb.CartServiceClient
	UserClient *grpc_client.UserClient
}

// ===================== LIST ======================
func (cc *CartController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cc.Client.ListCarts(ctx, &pb.ListCartRequest{
		OwnerId: userID,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Carts)
}

// ===================== GET ======================
func (cc *CartController) Get(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cc.Client.GetCart(ctx, &pb.GetCartRequest{
		Id:      uint32(id),
		OwnerId: userID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.PermissionDenied {
			return c.Status(403).JSON(fiber.Map{"error": "unauthorized"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Cart)
}

// ===================== CREATE ======================
func (cc *CartController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint32)

	var body struct {
		Products []struct {
			Id  uint32 `json:"id"`
			Qty uint32 `json:"qty"`
		} `json:"products"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// convert
	var items []*pb.CartProduct
	for _, p := range body.Products {
		items = append(items, &pb.CartProduct{
			Id:  p.Id,
			Qty: p.Qty,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cc.Client.CreateCart(ctx, &pb.CreateCartRequest{
		OwnerId:  userID,
		Products: items,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(resp.Cart)
}

// ===================== UPDATE ======================
func (cc *CartController) Update(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var body struct {
		Products []struct {
			Id  uint32 `json:"id"`
			Qty uint32 `json:"qty"`
		} `json:"products"`
		Status string `json:"status"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// convert
	var items []*pb.CartProduct
	for _, p := range body.Products {
		items = append(items, &pb.CartProduct{
			Id:  p.Id,
			Qty: p.Qty,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cc.Client.UpdateCart(ctx, &pb.UpdateCartRequest{
		Id:       uint32(id),
		Products: items,
		Status:   body.Status,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Cart)
}

// ===================== DELETE ======================
func (cc *CartController) Delete(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cc.Client.DeleteCart(ctx, &pb.DeleteCartRequest{
		Id:      uint32(id),
		OwnerId: userID,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}

// ===================== GET ALL (ADMIN) ======================
func (cc *CartController) GetAll(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cc.Client.GetAllCarts(ctx, &emptypb.Empty{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Carts)
}

// ===================== INIT ======================
func NewCartController() *CartController {
	conn, err := grpc.Dial("localhost:50055", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect to cart gRPC: " + err.Error())
	}

	client := pb.NewCartServiceClient(conn)
	userClient := grpc_client.NewUserClient()

	return &CartController{
		Client:     client,
		UserClient: userClient,
	}
}
