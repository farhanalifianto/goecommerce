package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type StockItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}
// Custom type to handle []StockItem as JSON in DB
type StockList []StockItem

func (s StockList) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StockList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

type Product struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc"`
	CreatedAt time.Time `json:"created_at"`
	Price     float64   `json:"price"`
	Stock     StockList `json:"stock" gorm:"type:jsonb"`
}
