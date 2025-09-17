package model

import "time"

type Product struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc"`
	OwnerID   uint      `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
}
