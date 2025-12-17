package main

import (
	"transaction-service/cache"
	"transaction-service/grpc_server"
	kafkax "transaction-service/kafka"
	"transaction-service/middleware"
	"transaction-service/model"
	pb "transaction-service/proto/transaction"

	"database/sql"
	"log"
	"net"
	"os"
	"transaction-service/routes"

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
	name := getEnv("DB_NAME", "addressdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect address db:", err)
	}

	// AutoMigrate untuk jaga-jaga tabel ada
	if err := DB.AutoMigrate(&model.Transaction{}); err != nil {
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

		routes.RegisterTransactionRoutes(app, DB, middleware.AuthMiddleware())

		log.Println("HTTP server running on port 3007")
		if err := app.Listen(":3007"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	// grpc
	go func() {
		listener, err := net.Listen("tcp", ":50056")
		if err != nil {
			log.Fatalf("failed to listen on port 50056: %v", err)
		}
		redisAddr := os.Getenv("REDIS_ADDR")
		rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
 	   })

		grpcServer := grpc.NewServer()
		TransactionServer := grpc_server.NewTransactionServer(SQLDB, producer, rdb)
		pb.RegisterTransactionServiceServer(grpcServer, TransactionServer)


		log.Println("gRPC server running on port 50052")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()
	consumer := kafkax.NewConsumer()

	consumer.Consume("payment.paid",kafkax.PaymentPaidHandler(SQLDB),)
	select {}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
