package main

import (
	"cart-service/cache"
	kafkax "cart-service/kafka"
	"cart-service/middleware"
	"cart-service/model"
	pb "cart-service/proto/cart"

	"cart-service/grpc_server"
	"cart-service/routes"
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

// 🟢 INIT DATABASE
func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "cartdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect cart db:", err)
	}

	// AutoMigrate untuk jaga-jaga tabel ada
	if err := DB.AutoMigrate(&model.Cart{}); err != nil {
		log.Fatal(err)
	}

	// 🟢 Ambil *sql.DB dari koneksi GORM
	SQLDB, err = DB.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB from gorm:", err)
	}
}



func main() {
	initDB()
	producer := kafkax.NewProducer()
	cache.ConnectRedis()

	// fiber
	go func() {
		app := fiber.New()
		app.Use(logger.New())

		routes.RegisterCartRoutes(app, DB, middleware.AuthMiddleware())

		log.Println("HTTP server running on port 3006")
		if err := app.Listen(":3006"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	// grpc
	go func() {
		listener, err := net.Listen("tcp", ":50055")
		if err != nil {
			log.Fatalf("failed to listen on port 50055: %v", err)
		}
		redisAddr := os.Getenv("REDIS_ADDR")
		rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
 	   })

		grpcServer := grpc.NewServer()
		cartServer := &grpc_server.CartServer{
			DB:            SQLDB,
			Producer: producer,
			Redis: rdb,
		}
		pb.RegisterCartServiceServer(grpcServer, cartServer)

		log.Println("gRPC server running on port 50055")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()
	consumer := kafkax.NewConsumer()
	consumer.Consume("cart.paid", kafkax.CartPaidHandler(DB))
	select {}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
