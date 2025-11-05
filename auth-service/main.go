package main

import (
	"auth-service/grpc_server"
	"auth-service/model"
	pb "auth-service/proto/auth"
	"auth-service/routes"
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

// --------------------
// Init Database
// --------------------
func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "authdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	if err := DB.AutoMigrate(&model.User{}); err != nil {
		log.Fatal("failed to migrate:", err)
	}

	log.Println("Connected to Auth DB:", name)
}

// --------------------
// Main Function
// --------------------
func main() {
	initDB()

	jwtSecret := getEnv("JWT_SECRET", "verysecretkey")

	// Jalankan HTTP (Fiber)
	go func() {
		app := fiber.New()
		app.Use(logger.New())

		// Tambahkan route utama
		app.Get("/", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"status":  "auth-service running",
				"version": "1.0.0",
			})
		})

		// ✅ Register semua endpoint auth
		routes.RegisterAuthRoutes(app)

		log.Println("HTTP server running on :3002")
		if err := app.Listen(":3002"); err != nil {
			log.Fatal(err)
		}
	}()

	// Jalankan gRPC
	go func() {
		listener, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatal("failed to listen:", err)
		}

		grpcServer := grpc.NewServer()
		pb.RegisterAuthServiceServer(grpcServer, &grpc_server.AuthServer{
			DB:        DB,
			JWTSecret: jwtSecret,
		})

		log.Println("gRPC server running on :50052")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("failed to serve gRPC:", err)
		}
	}()

	select {}
}

// --------------------
// Helper: getEnv
// --------------------
func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
