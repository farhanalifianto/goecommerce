package main

import (
	"fmt"
	"log"
	"os"

	"transaction-service/controller"
	"transaction-service/model"
	"transaction-service/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	db := initDB()

	app := fiber.New()
	app.Use(logger.New())

	productServiceURL := getEnv("PRODUCT_SERVICE_URL", "http://product-service:3002")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://user-service:3001")

	tc := &controller.TransactionController{
		DB:               db,
		ProductServiceURL: productServiceURL,
	}

	routes.SetupTransactionRoutes(app, tc, userServiceURL)

	port := getEnv("PORT", "3005")
	log.Println("transaction-service running on port", port)
	app.Listen(":" + port)
}

func initDB() *gorm.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "txn_db")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, pass, name, port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	if err := db.AutoMigrate(&model.Transaction{}); err != nil {
		log.Fatal("failed to migrate:", err)
	}
	return db
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
