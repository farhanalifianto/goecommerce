package controller

import (
	"encoding/json"
	"fmt"
	"time"

	"cart-service/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CartController struct {
	DB *gorm.DB
}

func (cc *CartController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type ProductInput struct {
		ProductID uint `json:"product_id"`
		Qty       uint `json:"qty"`
	}

	var input struct {
		Products []ProductInput `json:"products"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if len(input.Products) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "products cannot be empty"})
	}

	var existing model.Cart
	err := cc.DB.Where("owner_id = ? AND status = ?", userID, "unpaid").First(&existing).Error

	// ✅ jika sudah ada cart unpaid
	if err == nil {
		var existingProducts []ProductInput
		if err := json.Unmarshal(existing.Products, &existingProducts); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to decode existing products"})
		}

		// Gabungkan: update qty kalau ada, tambahkan kalau baru
		for _, newProd := range input.Products {
			found := false
			for i, oldProd := range existingProducts {
				if oldProd.ProductID == newProd.ProductID {
					existingProducts[i].Qty += newProd.Qty // tambah qty
					found = true
					break
				}
			}
			if !found {
				existingProducts = append(existingProducts, newProd)
			}
		}

		updatedJSON, _ := json.Marshal(existingProducts)
		existing.Products = datatypes.JSON(updatedJSON)
		existing.UpdatedAt = time.Now()

		if err := cc.DB.Save(&existing).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to update cart"})
		}
		return c.Status(200).JSON(existing)
	}

	if err != gorm.ErrRecordNotFound {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	// ✅ belum ada cart → buat baru
	newProductsJSON, _ := json.Marshal(input.Products)
	cart := model.Cart{
		OwnerID:   userID,
		Products:  datatypes.JSON(newProductsJSON),
		Status:    "unpaid",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := cc.DB.Create(&cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create cart"})
	}

	return c.Status(201).JSON(cart)
}
func (cc *CartController) GetCart(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var cart model.Cart
	err := cc.DB.Where("owner_id = ? AND status = ?", userID, "unpaid").First(&cart).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"error": "no active cart found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	// Decode JSONB products
	var products []map[string]interface{}
	if err := json.Unmarshal(cart.Products, &products); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to decode products"})
	}

	return c.JSON(fiber.Map{
		"id":         cart.ID,
		"owner_id":   cart.OwnerID,
		"status":     cart.Status,
		"products":   products,
		"created_at": cart.CreatedAt,
		"updated_at": cart.UpdatedAt,
	})
}

func (cc *CartController) DeleteCart(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	productIDParam := c.Params("id")

	var productID uint
	if _, err := fmt.Sscanf(productIDParam, "%d", &productID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid product id"})
	}

	// Cari cart unpaid milik user
	var cart model.Cart
	err := cc.DB.Where("owner_id = ? AND status = ?", userID, "unpaid").First(&cart).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"error": "no active cart found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	// Decode JSONB ke array
	var products []struct {
		ProductID uint `json:"product_id"`
		Qty       uint `json:"qty"`
	}
	if err := json.Unmarshal(cart.Products, &products); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to decode products"})
	}

	// Filter produk yang bukan yang mau dihapus
	newProducts := make([]struct {
		ProductID uint `json:"product_id"`
		Qty       uint `json:"qty"`
	}, 0)

	found := false
	for _, p := range products {
		if p.ProductID != productID {
			newProducts = append(newProducts, p)
		} else {
			found = true
		}
	}

	if !found {
		return c.Status(404).JSON(fiber.Map{"error": "product not found in cart"})
	}

	// Encode ulang dan simpan
	updatedJSON, _ := json.Marshal(newProducts)
	cart.Products = datatypes.JSON(updatedJSON)
	cart.UpdatedAt = time.Now()

	if err := cc.DB.Save(&cart).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update cart"})
	}

	return c.JSON(fiber.Map{
		"message": "product removed from cart",
		"cart":    cart,
	})
}
