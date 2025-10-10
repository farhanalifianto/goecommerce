package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wishlist-service/middleware"
	"wishlist-service/model"
	"wishlist-service/routes"
)

var DB *gorm.DB

func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "wishlistdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect cart db:", err)
	}

	if err := DB.AutoMigrate(&model.Wishlist{}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()

	app := fiber.New()
	app.Use(logger.New())

	// inject DB & middleware ke routes
	routes.RegisterWishlistRoutes(app, DB, middleware.AuthRequired)

	app.Listen(":3005")
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
