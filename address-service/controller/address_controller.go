package controller

import (
	"address-service/grpc_client"
	"address-service/model"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AddressController struct {
	DB         *gorm.DB
	UserClient *grpc_client.UserClient
}

// GET /addresses
func (ac *AddressController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var address []model.Address
	if err := ac.DB.Where("owner_id = ?", userID).Find(&address).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch addresses"})
	}	

	// Ambil email user sekali saja (efisien)
	UserClient := grpc_client.NewUserClient()
	userInfo, err := UserClient.GetUserEmail(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch user info"})
	}

	// Ubah owner_id jadi email di response
	response := []map[string]interface{}{}
	for _, addr := range address {
		response = append(response, map[string]interface{}{
			"id":         addr.ID,
			"name":       addr.Name,
			"desc":       addr.Desc,
			"owner_id":   userInfo.Email, // ✅ ubah ke email
			"created_at": addr.CreatedAt,
		})
	}

	return c.JSON(response)
}

// GET /addresses/:id
func (ac *AddressController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var address model.Address
	if err := ac.DB.Where("id = ? AND owner_id = ?", id, userID).First(&address).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "address not found"})
	}

	// Ambil email dari user-service via gRPC
	userInfo, err := ac.UserClient.GetUserEmail(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch user info"})
	}

	// Response dengan owner_id = email
	return c.JSON(fiber.Map{
		"id":         address.ID,
		"name":       address.Name,
		"desc":       address.Desc,
		"owner_id":   userInfo.Email, // ✅ email
		"created_at": address.CreatedAt,
	})
}

func (ac *AddressController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var in model.Address
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if in.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	in.OwnerID = userID
	in.CreatedAt = time.Now()

	if err := ac.DB.Create(&in).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create address"})
	}

	return c.Status(201).JSON(in)
}

func (ac *AddressController) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var address model.Address
	if err := ac.DB.Where("id = ? AND owner_id = ?", id, userID).First(&address).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "address not found"})
	}

	var input model.Address
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	address.Name = input.Name
	address.Desc = input.Desc

	if err := ac.DB.Save(&address).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update address"})
	}

	return c.JSON(address)
}

// Delete address (hanya jika milik user)
func (ac *AddressController) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	userID := c.Locals("user_id").(uint)

	var address model.Address
	if err := ac.DB.Where("id = ? AND owner_id = ?", id, userID).First(&address).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "address not found"})
	}

	if err := ac.DB.Delete(&address).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete address"})
	}

	return c.SendStatus(204)
}
