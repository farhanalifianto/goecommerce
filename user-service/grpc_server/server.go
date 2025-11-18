package grpc_server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"user-service/model"
	pb "user-service/proto/user"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	DB        *gorm.DB
	JWTSecret string
	Redis     *redis.Client
}

// =========================================================
// GET ME (with Redis cache)
// =========================================================
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

	cacheKey := fmt.Sprintf("user:me:%d", uint(sub))

	// ---- Try Redis HIT ----
	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var resp pb.UserResponse
		json.Unmarshal([]byte(cached), &resp)
		fmt.Println("🔥 Redis HIT (GetMe)")
		return &resp, nil
	}

	fmt.Println("❄ Redis MISS → DB (GetMe)")

	var user model.User
	if err := s.DB.First(&user, uint(sub)).Error; err != nil {
		return nil, err
	}

	resp := &pb.UserResponse{
		Id:    uint32(user.ID),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}

	// Save to Redis (TTL 5 minutes)
	jsonData, _ := json.Marshal(resp)
	s.Redis.Set(ctx, cacheKey, jsonData, 5*time.Minute)

	return resp, nil
}

// =========================================================
// GET USER INFO (with Redis cache)
// =========================================================
func (s *UserServer) GetUserInfo(ctx context.Context, in *pb.GetUserRequest) (*pb.UserResponse, error) {

	cacheKey := fmt.Sprintf("user:%d", in.Id)

	// ---- Try Redis HIT ----
	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var resp pb.UserResponse
		json.Unmarshal([]byte(cached), &resp)
		fmt.Println("🔥 Redis HIT (GetUserInfo)")
		return &resp, nil
	}

	fmt.Println("❄ Redis MISS → DB (GetUserInfo)")

	var user model.User
	if err := s.DB.First(&user, in.Id).Error; err != nil {
		return nil, err
	}

	resp := &pb.UserResponse{
		Id:    uint32(user.ID),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}

	// Save to Redis (TTL 5 min)
	jsonData, _ := json.Marshal(resp)
	s.Redis.Set(ctx, cacheKey, jsonData, 5*time.Minute)

	return resp, nil
}

// =========================================================
// GET USERS (cache all list)
// =========================================================
func (s *UserServer) GetUsers(ctx context.Context, _ *pb.Empty) (*pb.UsersResponse, error) {

	cacheKey := "users:all"

	// ---- Redis HIT ----
	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var resp pb.UsersResponse
		json.Unmarshal([]byte(cached), &resp)
		fmt.Println("🔥 Redis HIT (GetUsers)")
		return &resp, nil
	}

	fmt.Println("❄ Redis MISS → DB (GetUsers)")

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

	// Cache (TTL 30 seconds)
	jsonData, _ := json.Marshal(resp)
	s.Redis.Set(ctx, cacheKey, jsonData, 30*time.Second)

	return resp, nil
}
