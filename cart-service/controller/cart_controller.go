package controller

import (
	"strconv"
	"time"

	"cart-service/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CartController struct {
	DB *gorm.DB
}


func (cc *CartController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var Cart []model.Cart
	if err := cc.DB.Where("owner_id = ?", userID).Find(&Cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch Cart"})
	}

	return c.JSON(Cart)
}

func (cc *CartController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var cart model.Cart
	if err := cc.DB.Where("id = ? AND owner_id = ?", id, userID).First(&cart).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cart not found"})
	}

	return c.JSON(cart)
}

func (cc *CartController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var in model.Cart
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if in.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	in.OwnerID = userID
	in.CreatedAt = time.Now()

	if err := cc.DB.Create(&in).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create cart"})
	}

	return c.Status(201).JSON(in)
}

func (cc *CartController) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var cart model.Cart
	if err := cc.DB.Where("id = ? AND owner_id = ?", id, userID).First(&cart).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cart not found"})
	}

	var input model.Cart
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	cart.Name = input.Name
	cart.Desc = input.Desc

	if err := cc.DB.Save(&cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update cart"})
	}

	return c.JSON(cart)
}

// Delete cart (hanya jika milik user)
func (cc *CartController) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var cart model.Cart
	if err := cc.DB.Where("id = ? AND owner_id = ?", id, userID).First(&cart).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cart not found"})
	}

	if err := cc.DB.Delete(&cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete cart"})
	}

	return c.SendStatus(204)
}
