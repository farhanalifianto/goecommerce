package controller

import (
	"product-service/grpc_client"
	pb "product-service/proto/product"

	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductController struct {
	Client      pb.ProductServiceClient
	UserClient  *grpc_client.UserClient // kalau nanti mau get email buyer
}

// ===============================
//         LIST PRODUCTS
// ===============================
func (pc *ProductController) ListProducts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.ListProducts(ctx, &emptypb.Empty{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Products)
}

// ===============================
//         GET SINGLE
// ===============================
func (pc *ProductController) GetProduct(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.GetProduct(ctx, &pb.GetProductRequest{
		Id: uint32(id),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Product)
}

// ===============================
//         CREATE PRODUCT
// ===============================
func (pc *ProductController) CreateProduct(c *fiber.Ctx) error {
	var body struct {
		Name       string `json:"name"`
		Desc       string `json:"desc"`
		Price      uint32 `json:"price"`
		CategoryID uint32 `json:"category_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:       body.Name,
		Desc:       body.Desc,
		Price:      body.Price,
		CategoryId: body.CategoryID,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(resp.Product)
}

// ===============================
//         UPDATE PRODUCT
// ===============================
func (pc *ProductController) UpdateProduct(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Name       string `json:"name"`
		Desc       string `json:"desc"`
		Price      uint32 `json:"price"`
		CategoryID uint32 `json:"category_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.UpdateProduct(ctx, &pb.UpdateProductRequest{
		Id:         uint32(id),
		Name:       body.Name,
		Desc:       body.Desc,
		Price:      body.Price,
		CategoryId: body.CategoryID,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Product)
}

// ===============================
//         DELETE PRODUCT
// ===============================
func (pc *ProductController) DeleteProduct(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.DeleteProduct(ctx, &pb.DeleteProductRequest{
		Id: uint32(id),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}

// ===============================
//         CATEGORY
// ===============================
func (pc *ProductController) CreateCategory(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.CreateCategory(ctx, &pb.CreateCategoryRequest{
		Name: body.Name,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Category)
}

func (pc *ProductController) ListCategories(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.ListCategories(ctx, &emptypb.Empty{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Categories)
}

func (pc *ProductController) UpdateCategory(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.UpdateCategory(ctx, &pb.UpdateCategoryRequest{
		Id:   uint32(id),
		Name: body.Name,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Category)
}
func (pc *ProductController) DeleteCategory(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.DeleteCategory(ctx, &pb.DeleteCategoryRequest{
		Id: uint32(id),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}


// ===============================
//         STOCK
// ===============================
func (pc *ProductController) UpdateStock(c *fiber.Ctx) error {
	var body struct {
		ProductID uint32 `json:"product_id"`
		Quantity  int32  `json:"quantity"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.UpdateStock(ctx, &pb.UpdateStockRequest{
		ProductId: body.ProductID,
		Quantity:  body.Quantity,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Stock)
}

func (pc *ProductController) GetStock(c *fiber.Ctx) error {
	productID, err := strconv.Atoi(c.Params("product_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid product_id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.GetStock(ctx, &pb.GetStockRequest{
		ProductId: uint32(productID),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Stock)
}

// ===============================
//         INIT CONTROLLER
// ===============================
func NewProductController() *ProductController {
	conn, err := grpc.Dial("localhost:50054", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect to product gRPC: " + err.Error())
	}

	client := pb.NewProductServiceClient(conn)
	userClient := grpc_client.NewUserClient()

	return &ProductController{
		Client:     client,
		UserClient: userClient,
	}
}
