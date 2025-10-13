package model

import (
	"time"

	"gorm.io/datatypes"
)

type Wishlist struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	OwnerID   uint           `json:"owner_id"`
	Products  datatypes.JSON `gorm:"type:jsonb" json:"products"` // [{"product_id":2}, {"product_id":5}]
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
