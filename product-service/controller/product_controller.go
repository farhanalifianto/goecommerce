package controller

import (
	"strconv"

	"product-service/model"

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
	userID := c.Locals("user_id").(uint)
	in := model.Product{}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}
	in.OwnerID = userID
	if err := pc.DB.Create(&in).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "create failed"})
	}
	return c.JSON(in)
}

type UpdateProductRequest struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	Price float64 `json:"price"`
}

func (pc *ProductController) Update(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var p model.Product
	if err := pc.DB.First(&p, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	// if err := c.BodyParser(&p); err != nil {
	// 	return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	// } bahaya bisa diubah idnya coi
	var body UpdateProductRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if err := pc.DB.Save(&p).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "update failed"})
	}
	return c.JSON(p)
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

func (pc *ProductController) DecrementStock(c *fiber.Ctx) error {
	id := c.Params("id")

	var product model.Product
	if err := pc.DB.First(&product, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "product not found"})
	}

	var body struct {
		Qty int `json:"qty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if body.Qty <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "qty must be > 0"})
	}

	if product.Stock < body.Qty {
		return c.Status(400).JSON(fiber.Map{"error": "not enough stock"})
	}

	product.Stock -= body.Qty
	if err := pc.DB.Save(&product).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update stock"})
	}

	return c.JSON(fiber.Map{
		"message": "stock decremented",
		"stock":   product.Stock,
	})
}