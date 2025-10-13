package controller

import (
	"encoding/json"
	"strconv"
	"time"
	"wishlist-service/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WishlistController struct {
	DB *gorm.DB
}

func (wc *WishlistController) CreateOrUpdate(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type ProductInput struct {
		ProductID uint `json:"product_id"`
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

	// Ambil wishlist user
	var wishlist model.Wishlist
	err := wc.DB.Where("owner_id = ?", userID).First(&wishlist).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	// --- Parse existing products ---
	type ProductItem struct {
		ProductID uint `json:"product_id"`
	}
	var existingProducts []ProductItem

	if err == nil && len(wishlist.Products) > 0 {
		if err := json.Unmarshal(wishlist.Products, &existingProducts); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to parse existing wishlist"})
		}
	}

	// --- Gabungkan produk lama dan baru tanpa duplikat ---
	productMap := make(map[uint]bool)
	for _, p := range existingProducts {
		productMap[p.ProductID] = true
	}

	for _, p := range input.Products {
		if !productMap[p.ProductID] {
			existingProducts = append(existingProducts, ProductItem(p))
			productMap[p.ProductID] = true
		}
	}

	// Encode ke JSONB
	mergedJSON, err := json.Marshal(existingProducts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to encode products"})
	}

	// --- Simpan (insert/update) ---
	if err == gorm.ErrRecordNotFound {
		wishlist = model.Wishlist{
			OwnerID:   userID,
			Products:  datatypes.JSON(mergedJSON),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := wc.DB.Create(&wishlist).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to create wishlist"})
		}
	} else {
		wishlist.Products = datatypes.JSON(mergedJSON)
		wishlist.UpdatedAt = time.Now()
		if err := wc.DB.Save(&wishlist).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to update wishlist"})
		}
	}

	return c.Status(200).JSON(wishlist)
}


func (wc *WishlistController) Get(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var wishlist model.Wishlist
	if err := wc.DB.Where("owner_id = ?", userID).First(&wishlist).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"error": "wishlist not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	return c.Status(200).JSON(wishlist)
}
func (wc *WishlistController) DeleteProduct(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	productIDParam := c.Params("product_id")

	productID, err := strconv.Atoi(productIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid product_id"})
	}

	var wishlist model.Wishlist
	if err := wc.DB.Where("owner_id = ?", userID).First(&wishlist).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"error": "wishlist not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	// Struktur produk di JSONB
	type ProductItem struct {
		ProductID uint `json:"product_id"`
	}

	var products []ProductItem
	if err := json.Unmarshal(wishlist.Products, &products); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to parse products"})
	}

	// Filter produk yang tidak dihapus
	newProducts := make([]ProductItem, 0)
	for _, p := range products {
		if p.ProductID != uint(productID) {
			newProducts = append(newProducts, p)
		}
	}

	// Jika tidak ada perubahan (produk tidak ditemukan)
	if len(newProducts) == len(products) {
		return c.Status(404).JSON(fiber.Map{"error": "product not found in wishlist"})
	}

	// Encode kembali ke JSONB dan simpan
	updatedJSON, err := json.Marshal(newProducts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to encode updated products"})
	}

	wishlist.Products = datatypes.JSON(updatedJSON)
	wishlist.UpdatedAt = time.Now()

	if err := wc.DB.Save(&wishlist).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update wishlist"})
	}

	return c.Status(200).JSON(wishlist)
}
