package main

import (
	"address-service/cache"
	"address-service/grpc_server"
	kafkax "address-service/kafka"
	"address-service/middleware"
	"address-service/model"
	pb "address-service/proto/address"

	"address-service/routes"
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
	name := getEnv("DB_NAME", "addressdb")

	dsn := "host=" + host + " user=" + user + " password=" + pass + " dbname=" + name + " port=" + port + " sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect address db:", err)
	}

	// AutoMigrate untuk jaga-jaga tabel ada
	if err := DB.AutoMigrate(&model.Address{}); err != nil {
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

		routes.RegisterAddressRoutes(app, DB, middleware.AuthMiddleware())

		log.Println("🌐 HTTP server running on port 3003")
		if err := app.Listen(":3003"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	// grpc
	go func() {
		listener, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatalf("failed to listen on port 50052: %v", err)
		}
		redisAddr := os.Getenv("REDIS_ADDR")
		rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
 	   })

		grpcServer := grpc.NewServer()
		addressServer := &grpc_server.AddressServer{
			DB:            SQLDB,
			Producer: producer,
			Redis: rdb,
		}
		pb.RegisterAddressServiceServer(grpcServer, addressServer)

		log.Println("gRPC server running on port 50052")
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
