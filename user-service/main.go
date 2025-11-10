package main

import (
	"log"
	"net"
	"os"
	"user-service/grpc_server"
	kafka "user-service/kafka"
	"user-service/middleware"
	"user-service/model"
	pb "user-service/proto/user"
	"user-service/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// --------------------
// Database Init
// --------------------
func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "userdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect db:", err)
	}

	DB.AutoMigrate(&model.User{})
	log.Println("Connected to user database:", name)
}

// --------------------
// Main Function
// --------------------
func main() {
	initDB()

	jwtSecret := "verysecretkey"

	// Jalankan HTTP (Fiber)
	go func() {
		app := fiber.New()
		app.Use(logger.New())

		// Register routes dengan middleware Auth
		routes.RegisterUserRoutes(app, middleware.AuthMiddleware())

		log.Println("HTTP running on :3001")
		if err := app.Listen(":3001"); err != nil {
			log.Fatal(err)
		}
	}()

	// Jalankan gRPC Server
	go func() {
		listener, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatal(err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterUserServiceServer(grpcServer, &grpc_server.UserServer{DB: DB, JWTSecret: jwtSecret})

		log.Println("gRPC running on :50051")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	go kafka.StartUserCreatedConsumer(DB)

	select {}
}

// --------------------
// Helper
// --------------------
func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
