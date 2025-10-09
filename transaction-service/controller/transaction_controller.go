package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"transaction-service/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type TransactionController struct {
	DB               *gorm.DB
	ProductServiceURL string
	ProductService	  string
}

// CreateTransaction - membuat transaksi baru
func (tc *TransactionController) CreateTransaction(c *fiber.Ctx) error {
	var req struct {
			ProductID uint   `json:"product_id"`
			Variant   string `json:"variant"`
			Qty       int    `json:"qty"`
			AddressID uint   `json:"address_id"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		userID := c.Locals("user_id").(uint)

		// 1️⃣ Kurangi stok di product-service
		reduceURL := fmt.Sprintf("%s/api/products/%d/reduce", tc.ProductService, req.ProductID)
		payload, _ := json.Marshal(map[string]interface{}{
			"variant": req.Variant,
			"qty":     req.Qty,
		})

		resp, err := http.Post(reduceURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to connect to product-service"})
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			var errRes map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errRes)
			return c.Status(resp.StatusCode).JSON(fiber.Map{"error": errRes["error"]})
		}

		// 2️⃣ Ambil harga produk dari product-service
		productURL := fmt.Sprintf("%s/api/products/%d", tc.ProductService, req.ProductID)
		prodResp, err := http.Get(productURL)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch product"})
		}
		defer prodResp.Body.Close()

		if prodResp.StatusCode != 200 {
			return c.Status(prodResp.StatusCode).JSON(fiber.Map{"error": "product not found"})
		}

		var product struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Price float64 `json:"price"`
		}
		if err := json.NewDecoder(prodResp.Body).Decode(&product); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "invalid product response"})
		}

		// 3️⃣ Buat transaksi di DB
		transaction := model.Transaction{
			UserID:    userID,
			ProductID: req.ProductID,
			AddressID: req.AddressID,
			Qty:       req.Qty,
			Amount:    float64(req.Qty) * product.Price,
			Status:    "pending",
			CreatedAt: time.Now(),
		}

		if err := tc.DB.Create(&transaction).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to create transaction"})
		}

		return c.Status(201).JSON(transaction)
}


func (tc *TransactionController) GetUserTransactions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role := c.Locals("user_role").(string)

	var txns []model.Transaction
	if role == "admin" {
		tc.DB.Order("created_at desc").Find(&txns)
	} else {
		tc.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&txns)
	}
	return c.JSON(txns)
}


func (tc *TransactionController) GetTransactionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("user_role").(string)

	var txn model.Transaction
	if err := tc.DB.First(&txn, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	if txn.UserID != userID && role != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
	}

	return c.JSON(txn)
}
func (tc *TransactionController) GetTransactionsByUserID(c *fiber.Ctx) error {
    
    role := c.Locals("user_role").(string)
    if role != "admin" {
        return c.Status(403).JSON(fiber.Map{"error":"forbidden"})
    }
    uid := c.Params("user_id")
    var txns []model.Transaction
    tc.DB.Where("user_id = ?", uid).Order("created_at desc").Find(&txns)
    return c.JSON(txns)
}
