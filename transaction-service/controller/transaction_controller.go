package controller

import (
	"gorm.io/gorm"
)

type TransactionController struct {
	DB               *gorm.DB
	ProductServiceURL string
	ProductService	  string
}

// CreateTransaction - membuat transaksi baru
