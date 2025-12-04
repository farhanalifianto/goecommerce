package kafka

import (
	"encoding/json"
	"log"

	"cart-service/model"

	"gorm.io/gorm"
)

type CartPaidEvent struct {
    CartID   uint32 `json:"cart_id"`
    OwnerID  uint `json:"owner_id"`
    PaidAt   string `json:"paid_at"`
    Products []struct {
        ID  uint32 `json:"id"`
        Qty uint32 `json:"qty"`
    } `json:"products"`
}

func CartPaidHandler(db *gorm.DB) func([]byte) {
    return func(msg []byte) {
        log.Println(" Received cart.paid event")

        var event CartPaidEvent
        if err := json.Unmarshal(msg, &event); err != nil {
            log.Printf(" Failed to unmarshal event: %v", err)
            return
        }

        var cart model.Cart
        if err := db.First(&cart, event.CartID).Error; err != nil {
            log.Printf("Cart %d not found: %v", event.CartID, err)
            return
        }

        // Update status
        cart.Status = "paid"

        if err := db.Save(&cart).Error; err != nil {
            log.Printf("Failed to update cart %d to PAID: %v", cart.ID, err)
            return
        }

        log.Printf("Cart %d status updated to PAID", cart.ID)

        // OPTIONAL — create new empty cart for user
        newCart := model.Cart{
            OwnerID: event.OwnerID,
            Status:  "active",
            Products: []model.CartProduct{},
        }

        if err := db.Create(&newCart).Error; err != nil {
            log.Printf("Failed to create new cart for user %d: %v", event.OwnerID, err)
        } else {
            log.Printf("Created new empty cart %d for user %d", newCart.ID, event.OwnerID)
        }
    }
}
