package model

import "time"

type Payment struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `json:"transaction_id"` // relasi ke transaction
	UserID        uint      `json:"user_id"`        // biar gampang validasi owner
	Amount        int64     `json:"amount"`         // snapshot dari transaction.total_amount
	Status        string    `json:"status"`         // pending | paid | failed | expired
	Method        string    `json:"method"`         // manual | transfer | dummy
	CreatedAt     time.Time `json:"created_at"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
}
