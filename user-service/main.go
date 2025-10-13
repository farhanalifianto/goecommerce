package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"user-service/grpc_server"
	"user-service/model"

	pb "user-service/proto/user"
	"user-service/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
)

var DB *gorm.DB






func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASS", "postgres")
	name := getEnv("DB_NAME", "userdb")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", host, user, pass, name, port)
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect db:", err)
	}
	if err := DB.AutoMigrate(&model.User{}); err != nil {
		log.Fatal(err)
	}
}



func main() {
	initDB()
    jwtSecret := getEnv("JWT_SECRET", "verysecretkey")

    //fiber goroutines
    go func() {
        app := fiber.New()
        app.Use(logger.New())
        routes.RegisterUserRoutes(app, DB, jwtSecret)

        if err := app.Listen(":3001"); err != nil {
            log.Fatalf("failed to start user-service: %v", err)
        }
		
    }()

    // grpc main thread
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    grpcServer := grpc.NewServer()
    pb.RegisterUserServiceServer(grpcServer, &grpc_server.UserGRPCServer{DB: DB})
    log.Println("User gRPC running on :50051")
	reflection.Register(grpcServer)

    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("failed to serve gRPC: %v", err)
    }

}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
