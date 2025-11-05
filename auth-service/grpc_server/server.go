package grpc_server

import (
	"auth-service/model"
	pb "auth-service/proto/auth"
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	DB        *gorm.DB
	JWTSecret string
}

func (s *AuthServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.AuthResponse, error) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	user := model.User{
		Email:    in.Email,
		Password: string(hashed),
		Name:     in.Name,
		Role:     in.Role,
	}
	if err := s.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &pb.AuthResponse{
		Id:    uint32(user.ID),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	var user model.User
	if err := s.DB.Where("email = ?", in.Email).First(&user).Error; err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(s.JWTSecret))

	return &pb.LoginResponse{AccessToken: signed}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, in *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
    token, err := jwt.Parse(in.Token, func(token *jwt.Token) (interface{}, error) {
        return []byte(s.JWTSecret), nil
    })
    if err != nil || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, fmt.Errorf("invalid claims")
    }

    sub, ok := claims["sub"].(float64)
    if !ok {
        return nil, fmt.Errorf("invalid sub claim")
    }

    email, _ := claims["email"].(string)
    role, _ := claims["role"].(string)

    return &pb.ValidateTokenResponse{
        Id:    uint32(sub),
        Email: email,
        Role:  role, // ✅ pastikan ini ada
    }, nil
}