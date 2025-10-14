package main

import (
	"address-service/grpc_server"
	"address-service/middleware"
	"address-service/model"
	pb "address-service/proto/address"
	"address-service/routes"
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
	name := getEnv("DB_NAME", "addressdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect address db:", err)
	}

	if err := DB.AutoMigrate(&model.Address{}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()

	// Jalankan Fiber HTTP di goroutine
	go func() {
		app := fiber.New()
		app.Use(logger.New())

		// Inject DB & middleware ke routes
		routes.RegisterAddressRoutes(app, DB, middleware.AuthRequired)

		log.Println("🚀 HTTP server running on port 3003")
		if err := app.Listen(":3003"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	// Jalankan gRPC server di port berbeda
	go func() {
		listener, err := net.Listen("tcp", ":50052") // contoh port gRPC
		if err != nil {
			log.Fatalf("failed to listen on port 50052: %v", err)
		}

		grpcServer := grpc.NewServer()
		addressServer := &grpc_server.AddressServer{DB: DB}
		pb.RegisterAddressServiceServer(grpcServer, addressServer)

		log.Println("🛰️ gRPC server running on port 50052")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// block forever (biar main ga keluar)
	select {}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
