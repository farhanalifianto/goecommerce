package controller

import (
	"encoding/json"
	"product-service/model"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)



type ProductController struct {
	DB *gorm.DB
}

func (pc *ProductController) List(c *fiber.Ctx) error {
	var products model.Product
	pc.DB.Find(&products)
	return c.JSON(products)
}

func (pc *ProductController) Get(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var p model.Product
	if err := pc.DB.First(&p, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(p)
}

func (pc *ProductController) Create(c *fiber.Ctx) error {
	var in model.Product

	// Parse body ke struct Product
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Validasi dasar (optional)
	if in.Name == "" || len(in.Stock) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "name and stock are required"})
	}

	// Tambahkan created_at
	in.CreatedAt = time.Now()

	// Simpan ke DB
	if err := pc.DB.Create(&in).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "create failed"})
	}

	// Return response JSON
	return c.Status(201).JSON(in)
}

type UpdateProductRequest struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	Price float64 `json:"price"`
}

func (pc *ProductController) Update(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var product model.Product
	if err := pc.DB.First(&product, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "product not found"})
	}

	// Parsing input body ke struct baru
	var input model.Product
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Update field-field yang diizinkan
	product.Name = input.Name
	product.Desc = input.Desc
	product.Price = input.Price
	product.Stock = input.Stock // ← ini sudah []StockItem / StockList

	// Simpan ke DB
	if err := pc.DB.Save(&product).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "update failed"})
	}

	// Return hasil update
	return c.JSON(product)
}

func (pc *ProductController) Delete(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var p model.Product
	if err := pc.DB.First(&p, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if err := pc.DB.Delete(&p).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "delete failed"})
	}
	return c.SendStatus(204)
}

func (pc *ProductController) ReduceStock(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Variant string `json:"variant"`
		Qty     int    `json:"qty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	var p model.Product
	if err := pc.DB.First(&p, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "product not found"})
	}

	stocks := p.Stock

	found := false
	for i := range stocks {
		if stocks[i].Name == req.Variant {
			if stocks[i].Qty < req.Qty {
				return c.Status(400).JSON(fiber.Map{"error": "not enough stock"})
			}
			stocks[i].Qty -= req.Qty
			found = true
			break
		}
	}
	if !found {
		return c.Status(404).JSON(fiber.Map{"error": "variant not found"})
	}

	updated, _ := json.Marshal(stocks)
	pc.DB.Model(&p).Update("stock", updated)
	return c.JSON(fiber.Map{"success": true})
}