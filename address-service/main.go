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
	"time"

	"github.com/IBM/sarama"
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

func initKafkaProducer() sarama.SyncProducer {
	brokers := []string{getEnv("KAFKA_BROKER", "kafka:9092")}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll

	var producer sarama.SyncProducer
	var err error

	for i := 1; i <= 10; i++ {
		producer, err = sarama.NewSyncProducer(brokers, config)
		if err == nil {
			log.Println("✅ Connected to Kafka broker", brokers)
			return producer
		}
		log.Printf("⏳ Waiting for Kafka... (%d/10) Error: %v", i, err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf("❌ Could not connect to Kafka after retries: %v", err)
	return nil
}

func main() {
	initDB()
	producer := initKafkaProducer()

	go func() {
		app := fiber.New()
		app.Use(logger.New())

		routes.RegisterAddressRoutes(app, DB, middleware.AuthMiddleware())

		log.Println("HTTP server running on port 3003")
		if err := app.Listen(":3003"); err != nil {
			log.Fatal("fiber error:", err)
		}
	}()

	
	go func() {
		listener, err := net.Listen("tcp", ":50052") 
		if err != nil {
			log.Fatalf("failed to listen on port 50052: %v", err)
		}

		grpcServer := grpc.NewServer()
		addressServer := &grpc_server.AddressServer{DB: DB,KafkaProducer: producer,}
		pb.RegisterAddressServiceServer(grpcServer, addressServer)

		log.Println("🛰️ gRPC server running on port 50052")
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
