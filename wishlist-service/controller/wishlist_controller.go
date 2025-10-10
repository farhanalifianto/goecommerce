package controller

import (
	"strconv"
	"time"

	"wishlist-service/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type WishlistController struct {
	DB *gorm.DB
}


func (wc *WishlistController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var Wishlist []model.Wishlist
	if err := wc.DB.Where("owner_id = ?", userID).Find(&Wishlist).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch Wishlist"})
	}

	return c.JSON(Wishlist)
}

func (wc *WishlistController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var wishlist model.Wishlist
	if err := wc.DB.Where("id = ? AND owner_id = ?", id, userID).First(&wishlist).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cart not found"})
	}

	return c.JSON(wishlist)
}

func (wc *WishlistController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var in model.Wishlist
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if in.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	in.OwnerID = userID
	in.CreatedAt = time.Now()

	if err := wc.DB.Create(&in).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create cart"})
	}

	return c.Status(201).JSON(in)
}

func (wc *WishlistController) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var cart model.Wishlist
	if err := wc.DB.Where("id = ? AND owner_id = ?", id, userID).First(&cart).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cart not found"})
	}

	var input model.Wishlist
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	cart.Name = input.Name
	cart.Desc = input.Desc

	if err := wc.DB.Save(&cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update cart"})
	}

	return c.JSON(cart)
}

// Delete cart (hanya jika milik user)
func (wc *WishlistController) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var cart model.Wishlist
	if err := wc.DB.Where("id = ? AND owner_id = ?", id, userID).First(&cart).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cart not found"})
	}

	if err := wc.DB.Delete(&cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete cart"})
	}

	return c.SendStatus(204)
}
