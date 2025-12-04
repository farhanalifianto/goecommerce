package model

import (
	"time"
)

type Cart struct {
    ID        uint          `gorm:"primaryKey" json:"id"`
    OwnerID   uint          `json:"owner_id"`
    Products  []CartProduct `gorm:"type:json" json:"products"`
    Status    string        `json:"status"` // active / paid
    CreatedAt time.Time     `json:"created_at"`
}

type CartProduct struct {
    ID  uint `json:"id"`
    Qty uint `json:"qty"`
}
