package grpc_server

import (
	"context"
	"fmt"
	"time"
	"user-service/model"
	pb "user-service/proto/user"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	DB        *gorm.DB
	JWTSecret string
}

func (s *UserServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.UserResponse, error) {
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

	return &pb.UserResponse{
		Id:    uint32(user.ID),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
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

func (s *UserServer) GetMe(ctx context.Context, in *pb.GetMeRequest) (*pb.UserResponse, error) {
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

	var user model.User
	if err := s.DB.First(&user, uint(sub)).Error; err != nil {
		return nil, err
	}

	return &pb.UserResponse{
		Id:    uint32(user.ID),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}
func (s *UserServer) GetUsers(ctx context.Context, _ *pb.Empty) (*pb.UsersResponse, error) {
	var users []model.User
	if err := s.DB.Select("id", "email", "name", "role").Find(&users).Error; err != nil {
		return nil, err
	}

	resp := &pb.UsersResponse{}
	for _, u := range users {
		resp.Users = append(resp.Users, &pb.UserResponse{
			Id:    uint32(u.ID),
			Email: u.Email,
			Name:  u.Name,
			Role:  u.Role,
		})
	}
	return resp, nil
}
