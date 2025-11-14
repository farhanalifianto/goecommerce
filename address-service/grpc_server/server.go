package grpc_server

import (
	"context"
	"database/sql"
	"time"

	kafka "address-service/kafka"
	"address-service/model"
	pb "address-service/proto/address"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AddressServer struct {
	pb.UnimplementedAddressServiceServer
	DB            *sql.DB
	Producer *kafka.Producer
}

func (s *AddressServer) CreateAddress(ctx context.Context, req *pb.CreateAddressRequest) (*pb.AddressResponse, error) {
	query := `INSERT INTO addresses (name, "desc", owner_id, created_at) VALUES ($1, $2, $3, NOW()) RETURNING id`
	var id uint32

	err := s.DB.QueryRowContext(ctx, query, req.Name, req.Desc, req.OwnerId).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert: %v", err)
	}

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


func (s *AddressServer) GetAddress(ctx context.Context, in *pb.GetAddressRequest) (*pb.AddressResponse, error) {
	query := `SELECT id, name, "desc", owner_id, created_at FROM addresses WHERE id = $1`
	row := s.DB.QueryRowContext(ctx, query, in.Id)

	var a model.Address
	err := row.Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "address not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// Validasi owner
	if a.OwnerID != uint(in.OwnerId) {
		return nil, status.Errorf(codes.PermissionDenied, "unauthorized: not your address")
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


func (s *AddressServer) ListAddresses(ctx context.Context, req *pb.ListAddressRequest) (*pb.ListAddressResponse, error) {
	query := `SELECT id, name, "desc", owner_id, created_at FROM addresses WHERE owner_id = $1`
	rows, err := s.DB.QueryContext(ctx, query, req.OwnerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var addresses []*pb.Address
	for rows.Next() {
		var a model.Address
		err := rows.Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt)
		if err != nil {
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

	return &pb.ListAddressResponse{Addresses: addresses}, nil
}


func (s *AddressServer) UpdateAddress(ctx context.Context, req *pb.UpdateAddressRequest) (*pb.AddressResponse, error) {
	query := `UPDATE addresses SET name=$1, "desc"=$2 WHERE id=$3 RETURNING id, name, "desc", owner_id, created_at`
	row := s.DB.QueryRowContext(ctx, query, req.Name, req.Desc, req.Id)

	var a model.Address
	err := row.Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "address not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

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


	return &pb.AddressResponse{Address: &pb.Address{
		Id:        uint32(a.ID),
		Name:      a.Name,
		Desc:      a.Desc,
		OwnerId:   uint32(a.OwnerID),
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}}, nil
}

// 🟢 DELETE
func (s *AddressServer) DeleteAddress(ctx context.Context, req *pb.DeleteAddressRequest) (*pb.DeleteAddressResponse, error) {
	query := `DELETE FROM addresses WHERE id=$1`
	res, err := s.DB.ExecContext(ctx, query, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete error: %v", err)
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		return nil, status.Errorf(codes.NotFound, "address not found")
	}
	event := map[string]interface{}{
	"event_type": "address_deleted",
	"data": map[string]interface{}{
		"id": req.Id,
		},
	}

s.Producer.PublishAddressDeletedEvent(event)


	return &pb.DeleteAddressResponse{Message: "Address deleted successfully"}, nil
}

// 🟢 GET ALL
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
		err := rows.Scan(&a.ID, &a.Name, &a.Desc, &a.OwnerID, &a.CreatedAt)
		if err != nil {
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
