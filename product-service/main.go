package main

import (
	"product-service/cache"
	"product-service/grpc_server"
	kafkax "product-service/kafka"
	"product-service/middleware"
	"product-service/model"
	pb "product-service/proto/product"
	"product-service/routes"

	"database/sql"
	"log"
	"net"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var SQLDB *sql.DB

// INIT DATABASE
func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "productdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + 
		" dbname=" + name + " port=" + port + 
		" sslmode=disable TimeZone=UTC"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect product db:", err)
	}

	// AutoMigrate models
	if err := DB.AutoMigrate(&model.Product{}, &model.Category{}, &model.Stock{}); err != nil {
		log.Fatal(err)
	}

	SQLDB, err = DB.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB:", err)
	}
}

func main() {
	initDB()
	producer := kafkax.NewProducer()
	cache.ConnectRedis()

	// HTTP SERVER (Fiber)
	go func() {
		app := fiber.New()
		app.Use(logger.New())

		// Register product routes
		routes.RegisterProductRoutes(app, DB, middleware.AuthMiddleware())

		log.Println("HTTP product server running on port 3005")
		if err := app.Listen(":3005"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	// gRPC SERVER
	go func() {
		listener, err := net.Listen("tcp", ":50054")
		if err != nil {
			log.Fatalf("failed to listen on port 50054: %v", err)
		}

		redisAddr := os.Getenv("REDIS_ADDR")
		rdb := redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})

		grpcServer := grpc.NewServer()
		productServer := &grpc_server.ProductServer{
			DB:       DB,
			Producer: producer,
			Redis:    rdb,
		}

		pb.RegisterProductServiceServer(grpcServer, productServer)

		log.Println("gRPC product server running on port 50054")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
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
