package grpc_server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	kafka "address-service/kafka"
	"address-service/model"
	pb "address-service/proto/address"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AddressServer struct {
    pb.UnimplementedAddressServiceServer
    DB       *sql.DB
    Producer *kafka.Producer
    Redis    *redis.Client
}

// CREATE

func (s *AddressServer) CreateAddress(ctx context.Context, req *pb.CreateAddressRequest) (*pb.AddressResponse, error) {
    query := `INSERT INTO addresses (name, "desc", owner_id, created_at)
              VALUES ($1, $2, $3, NOW()) RETURNING id`

    var id uint32
    err := s.DB.QueryRowContext(ctx, query, req.Name, req.Desc, req.OwnerId).Scan(&id)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to insert: %v", err)
    }

    // Hapus cache LIST user ini
    cacheKey := fmt.Sprintf("addresses:%d", req.OwnerId)
    s.Redis.Del(ctx, cacheKey)

    // Event Kafka
    event := map[string]interface{}{
        "event_type": "address_created",
        "data": map[string]interface{}{
            "id":       id,
            "name":     req.Name,
            "desc":     req.Desc,
            "owner_id": req.OwnerId,
        },
    }
    s.Producer.PublishAddressCreatedEvent(event)

    return &pb.AddressResponse{
        Address: &pb.Address{
            Id:      id,
            Name:    req.Name,
            Desc:    req.Desc,
            OwnerId: req.OwnerId,
        },
    }, nil
}

// GET SINGLE

func (s *AddressServer) GetAddress(ctx context.Context, req *pb.GetAddressRequest) (*pb.AddressResponse, error) {
    query := `SELECT id, name, "desc", owner_id, created_at
              FROM addresses WHERE id = $1`

    var a model.Address
    err := s.DB.QueryRowContext(ctx, query, req.Id).
        Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt)

    if err == sql.ErrNoRows {
        return nil, status.Errorf(codes.NotFound, "address not found")
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query error: %v", err)
    }

    if a.OwnerID != uint(req.OwnerId) {
        return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
    }

    return &pb.AddressResponse{
        Address: &pb.Address{
            Id:        uint32(a.ID),
            Name:      a.Name,
            Desc:      a.Desc,
            OwnerId:   uint32(a.OwnerID),
            CreatedAt: a.CreatedAt.Format(time.RFC3339),
        },
    }, nil
}

// LIST (WITH REDIS CACHE)

func (s *AddressServer) ListAddresses(ctx context.Context, req *pb.ListAddressRequest) (*pb.ListAddressResponse, error) {
    cacheKey := fmt.Sprintf("addresses:%d", req.OwnerId)

    // 1. Check Redis
    cached, err := s.Redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var addresses []*pb.Address
        json.Unmarshal([]byte(cached), &addresses)
        fmt.Println("🔥 Redis HIT")
        return &pb.ListAddressResponse{Addresses: addresses}, nil
    }

    // Redis error tapi bukan key missing
    if err != redis.Nil {
        fmt.Println("Redis ERROR (bypass ke DB):", err)
    } else {
        fmt.Println("Redis MISS → DB query")
    }

    // 2. Query DB
    query := `SELECT id, name, "desc", owner_id, created_at
              FROM addresses WHERE owner_id = $1`

    rows, err := s.DB.QueryContext(ctx, query, req.OwnerId)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query error: %v", err)
    }
    defer rows.Close()

    var addresses []*pb.Address
    for rows.Next() {
        var a model.Address
        if err := rows.Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt); err != nil {
            return nil, status.Errorf(codes.Internal, "scan error: %v", err)
        }
        addresses = append(addresses, &pb.Address{
            Id:        uint32(a.ID),
            Name:      a.Name,
            Desc:      a.Desc,
            OwnerId:   uint32(a.OwnerID),
            CreatedAt: a.CreatedAt.Format(time.RFC3339),
        })
    }

    // 3. Save to Redis dengan TTL 5 menit
    jsonData, _ := json.Marshal(addresses)
    err = s.Redis.Set(ctx, cacheKey, jsonData, 5*time.Minute).Err()
    if err != nil {
        fmt.Println("⚠ Gagal set Redis:", err)
    }

    return &pb.ListAddressResponse{Addresses: addresses}, nil
}
//  UPDATE

func (s *AddressServer) UpdateAddress(ctx context.Context, req *pb.UpdateAddressRequest) (*pb.AddressResponse, error) {
    query := `UPDATE addresses SET name=$1, "desc"=$2
              WHERE id=$3 RETURNING id, name, "desc", owner_id, created_at`

    var a model.Address
    err := s.DB.QueryRowContext(ctx, query, req.Name, req.Desc, req.Id).
        Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt)

    if err == sql.ErrNoRows {
        return nil, status.Errorf(codes.NotFound, "address not found")
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "update error: %v", err)
    }

    // DELETE CACHE
    cacheKey := fmt.Sprintf("addresses:%d", a.OwnerID)
    s.Redis.Del(ctx, cacheKey)

    event := map[string]interface{}{
        "event_type": "address_updated",
        "data": map[string]interface{}{
            "id":       a.ID,
            "name":     a.Name,
            "desc":     a.Desc,
            "owner_id": a.OwnerID,
        },
    }
    s.Producer.PublishAddressUpdatedEvent(event)

    return &pb.AddressResponse{
        Address: &pb.Address{
            Id:        uint32(a.ID),
            Name:      a.Name,
            Desc:      a.Desc,
            OwnerId:   uint32(a.OwnerID),
            CreatedAt: a.CreatedAt.Format(time.RFC3339),
        },
    }, nil
}

//  DELETE

func (s *AddressServer) DeleteAddress(ctx context.Context, req *pb.DeleteAddressRequest) (*pb.DeleteAddressResponse, error) {
    // 1. Ambil owner_id address
    var ownerID uint32
    err := s.DB.QueryRowContext(ctx,
        `SELECT owner_id FROM addresses WHERE id=$1`, req.Id).
        Scan(&ownerID)

    if err == sql.ErrNoRows {
        return nil, status.Errorf(codes.NotFound, "address not found")
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query error: %v", err)
    }

    // 2.user yg sama dengan address yang akan dihapus
    if ownerID != req.OwnerId {
        return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
    }

    // 3. Delete
    _, err = s.DB.ExecContext(ctx,
        `DELETE FROM addresses WHERE id=$1`, req.Id)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "delete error: %v", err)
    }

    // 4. Bersihkan cache LIST address user ini
    cacheKey := fmt.Sprintf("addresses:%d", ownerID)
    s.Redis.Del(ctx, cacheKey)

    // 5. Publish event
    event := map[string]interface{}{
        "event_type": "address_deleted",
        "data": map[string]interface{}{
            "id":       req.Id,
            "owner_id": ownerID,
        },
    }
    s.Producer.PublishAddressDeletedEvent(event)

    return &pb.DeleteAddressResponse{
        Message: "Address deleted successfully",
    }, nil
}


// GET ALL (no cache)

func (s *AddressServer) GetAllAddresses(ctx context.Context, _ *emptypb.Empty) (*pb.GetAllAddressesResponse, error) {
    query := `SELECT id, name, "desc", owner_id, created_at FROM addresses`

    rows, err := s.DB.QueryContext(ctx, query)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query error: %v", err)
    }
    defer rows.Close()

    var addresses []*pb.Address
    for rows.Next() {
        var a model.Address
        if err := rows.Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt); err != nil {
            return nil, status.Errorf(codes.Internal, "scan error: %v", err)
        }
        addresses = append(addresses, &pb.Address{
            Id:        uint32(a.ID),
            Name:      a.Name,
            Desc:      a.Desc,
            OwnerId:   uint32(a.OwnerID),
            CreatedAt: a.CreatedAt.Format(time.RFC3339),
        })
    }

    return &pb.GetAllAddressesResponse{Addresses: addresses}, nil
}
