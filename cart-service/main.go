package main

import (
	"cart-service/grpc_server"
	"cart-service/middleware"
	"cart-service/model"
	pb "cart-service/proto/cart"
	"cart-service/routes"
	"log"
	"net"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "cartdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass +
		" dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect cart db:", err)
	}

	DB.AutoMigrate(&model.Cart{})
}

func main() {
	initDB()

	go func() {
		app := fiber.New()
		app.Use(logger.New())

		routes.RegisterCartRoutes(app, middleware.AuthRequired)

		log.Println("🚀 HTTP server running on port 3004")
		if err := app.Listen(":3004"); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatal(err)
		}
		s := grpc.NewServer()
		cartServer := &grpc_server.CartServer{DB: DB}
		pb.RegisterCartServiceServer(s, cartServer)

		log.Println("🛰️ gRPC server running on port 50052")
		if err := s.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	select {}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
