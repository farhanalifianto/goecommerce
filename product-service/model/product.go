package model

import (
	"time"
)

type Product struct {
    ID         uint      `gorm:"primaryKey" json:"id"`
    Name       string    `json:"name"`
    Desc       string    `json:"desc"`
    Price      uint      `json:"price"`
    CategoryID uint      `json:"category_id"`
    CreatedAt  time.Time `json:"created_at"`
}

type Category struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
}

type Stock struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    ProductID uint      `json:"product_id"`
    Quantity  int       `json:"quantity"`
    UpdatedAt time.Time `json:"updated_at"`
}
