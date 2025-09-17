package main

import (
	"fmt"
	"log"
	"os"

	"user-service/model"
	"user-service/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "userdb")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", host, user, pass, name, port)
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect db:", err)
	}
	if err := DB.AutoMigrate(&model.User{}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	jwtSecret := getEnv("JWT_SECRET", "secret")

	app := fiber.New()
	app.Use(logger.New())

	routes.RegisterUserRoutes(app, DB, jwtSecret)

	app.Listen(":3001")
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
