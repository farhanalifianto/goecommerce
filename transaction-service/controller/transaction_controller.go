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
}

// CreateTransaction - membuat transaksi baru
func (tc *TransactionController) CreateTransaction(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var body struct {
		ProductID uint    `json:"product_id"`
		Qty       int     `json:"qty"`
		Amount    float64 `json:"amount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}
	if body.Qty <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "qty must be > 0"})
	}

	// Panggil product-service untuk decrement stock
	decReq := map[string]int{"qty": body.Qty}
	decBody, _ := json.Marshal(decReq)
	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/products/%d/decrement", tc.ProductServiceURL, body.ProductID),
		bytes.NewReader(decBody),
	)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return c.Status(400).JSON(fiber.Map{"error": "failed to reserve stock"})
	}
	defer resp.Body.Close()

	
	txn := model.Transaction{
		UserID:    userID,
		ProductID: body.ProductID,
		Qty:       body.Qty,
		Amount:    body.Amount,
		Status:    "created",
		CreatedAt: time.Now(),
	}
	if err := tc.DB.Create(&txn).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save transaction"})
	}

	return c.Status(201).JSON(txn)
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
