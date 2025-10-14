package grpc_server

import (
	"address-service/model"
	pb "address-service/proto/address"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// Struct utama server gRPC
type AddressServer struct {
	pb.UnimplementedAddressServiceServer
	DB *gorm.DB
}

// GetAddressByID mencari address berdasarkan ID
func (s *AddressServer) GetAddressByID(ctx context.Context, req *pb.GetAddressRequest) (*pb.GetAddressResponse, error) {
	var address model.Address

	// Query ke database
	if err := s.DB.First(&address, req.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "address not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	// Return hasil
	return &pb.GetAddressResponse{
		Id:      uint32(address.ID),
		Name:    address.Name,
		Desc:    address.Desc,
		Ownerid: uint32(address.OwnerID), 
	}, nil
}
