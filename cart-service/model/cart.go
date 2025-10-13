package model

import (
	"time"

	"gorm.io/datatypes"
)

type Cart struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	OwnerID   uint           `json:"owner_id"`
	Products  datatypes.JSON `gorm:"type:jsonb" json:"products"`
	Status    string         `gorm:"type:varchar(20);default:'unpaid'" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
