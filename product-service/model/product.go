package model

import (
	"time"
)



type Product struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc"`
	CreatedAt time.Time `json:"created_at"`
	Price     float64   `json:"price"`
	Stock     float64   `json:"stock"`
}