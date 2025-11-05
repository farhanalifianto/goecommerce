package grpc_server

import (
	"context"
	"fmt"
	"user-service/model"
	pb "user-service/proto/user"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	DB        *gorm.DB
	JWTSecret string
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

func (s *UserServer) GetUserInfo(ctx context.Context, in *pb.GetUserRequest) (*pb.UserResponse, error) {
    var user model.User
    if err := s.DB.First(&user, in.Id).Error; err != nil {
        return nil, err
    }

    return &pb.UserResponse{
        Id:    uint32(user.ID),
        Email: user.Email,
        Name:  user.Name,
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

