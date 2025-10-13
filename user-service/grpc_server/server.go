package grpc_server

import (
	"context"
	"fmt"
	"user-service/model"
	pb "user-service/proto/user"

	"errors"
	"log"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type UserGRPCServer struct {
	pb.UnimplementedUserServiceServer
	DB *gorm.DB
}


var jwtSecret = []byte("verysecretkey")

type Claims struct {
	UserID uint `json:"sub"`
	jwt.RegisteredClaims
}
func (s *UserGRPCServer) ValidateToken(tokenString string) (uint, error) {
	fmt.Println("Validating token:", tokenString)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		fmt.Println("JWT parse error:", err)
		return 0, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		fmt.Println("Token claims cast failed")
		return 0, errors.New("invalid token claims")
	}
	if !token.Valid {
		fmt.Println("Token is not valid")
		return 0, errors.New("invalid token")
	}

	fmt.Println("Token OK - userID:", claims.UserID)
	return claims.UserID, nil
}

func (s *UserGRPCServer) GetUserByID(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    var user model.User
    if err := s.DB.First(&user, req.Id).Error; err != nil {
        log.Printf("User not found: %v", err)  // tambahin log
        return nil, err
    }

    return &pb.GetUserResponse{
        Id:    uint32(user.ID),
        Email: user.Email,
        Name:  user.Name,
        Role:  user.Role,
    }, nil
}

func (s *UserGRPCServer) GetMe(ctx context.Context, req *pb.GetMeRequest) (*pb.GetUserResponse, error) {
	userID, err := s.ValidateToken(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	var user model.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found by grpc")
	}

	return &pb.GetUserResponse{
		Id:    uint32(user.ID),
		Email: user.Email,
		Name:  user.Name,  
		Role:  user.Role,
	}, nil
}
