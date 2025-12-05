package model

import (
	"encoding/json"
	"time"
)

type Transaction struct {
    ID            uint      `gorm:"primaryKey"`

    UserID        uint
    CartID        uint

    AddressSnapshot  json.RawMessage `gorm:"type:jsonb"`
    ProductSnapshot  json.RawMessage `gorm:"type:jsonb"`

    TotalAmount   int64
    Status        string
    CreatedAt     time.Time
    PaidAt        *time.Time
}
