package kafka

import (
	"context"
	"encoding/json"
	"log"

	"gorm.io/gorm"
)

type CartCheckedOutEvent struct {
	EventType string `json:"event_type"`
	Data      struct {
		CartID        uint32 `json:"cart_id"`
		UserID        uint32 `json:"user_id"`
		TransactionID uint32 `json:"transaction_id"`
		TotalAmount   int64  `json:"total_amount"`
		CheckedOutAt  string `json:"checked_out_at"`
	} `json:"data"`
}

type CartEventHandler struct {
	DB *gorm.DB
}

func NewCartEventHandler(db *gorm.DB) *CartEventHandler {
	return &CartEventHandler{DB: db}
}

func (h *CartEventHandler) HandleCartCheckedOut(msg []byte) {
	var event CartCheckedOutEvent

	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal cart.paid event: %v", err)
		return
	}

	log.Printf("cart.paid received: cart_id=%d user_id=%d",
		event.Data.CartID,
		event.Data.UserID,
	)

	// idempotent update
	res := h.DB.WithContext(context.Background()).
		Table("carts").
		Where("id = ? AND owner_id = ? AND status = ?", event.Data.CartID, event.Data.UserID, "active").
		Update("status", "paid")

	if res.Error != nil {
		log.Printf("Failed to update cart status: %v", res.Error)
		return
	}

	if res.RowsAffected == 0 {
		log.Printf("Cart already checked out or not active (cart_id=%d)", event.Data.CartID)
		return
	}

	log.Printf("Cart %d marked as paid", event.Data.CartID)
}
