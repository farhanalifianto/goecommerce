package main

import (
	"payment-service/cache"
	"payment-service/grpc_server"
	kafkax "payment-service/kafka"
	"payment-service/middleware"
	"payment-service/model"
	pb "payment-service/proto/payment"
	"payment-service/routes"

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

var (
	DB    *gorm.DB
	SQLDB *sql.DB
)


// ======================
// INIT DATABASE
// ======================
func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "paymentdb")

	dsn := "host=" + host +
		" user=" + user +
		" password=" + pass +
		" dbname=" + name +
		" port=" + port +
		" sslmode=disable TimeZone=UTC"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect payment db:", err)
	}

	// Auto migrate
	if err := DB.AutoMigrate(&model.Payment{}); err != nil {
		log.Fatal(err)
	}

	// ambil *sql.DB
	SQLDB, err = DB.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB from gorm:", err)
	}
}

func main() {
	initDB()

	// kafka producer
	producer := kafkax.NewProducer()

	// redis
	cache.ConnectRedis()

	// ======================
	// HTTP SERVER (Fiber)
	// ======================
	go func() {
		app := fiber.New()
		app.Use(logger.New())

		routes.RegisterPaymentRoutes(
			app,
			middleware.AuthMiddleware(),
		)

		log.Println("HTTP server running on port 3004")
		if err := app.Listen(":3008"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	// ======================
	// gRPC SERVER
	// ======================
	go func() {
		listener, err := net.Listen("tcp", ":50057")
		if err != nil {
			log.Fatalf("failed to listen on port 50057: %v", err)
		}

		redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
		rdb := redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})

		grpcServer := grpc.NewServer()

		paymentServer := grpc_server.NewPaymentServer(
			SQLDB,
			rdb,
			producer,
		)

		pb.RegisterPaymentServiceServer(grpcServer, paymentServer)

		log.Println("gRPC server running on port 50054")
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
