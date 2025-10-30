package grpc_server

import (
	"address-service/model"
	pb "address-service/proto/address"
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type AddressServer struct {
	pb.UnimplementedAddressServiceServer
	DB *gorm.DB
}

func (s *AddressServer) CreateAddress(ctx context.Context, req *pb.CreateAddressRequest) (*pb.AddressResponse, error) {
	address := model.Address{
		Name:      req.Name,
		Desc:      req.Desc,
		OwnerID:   uint(req.OwnerId),
		CreatedAt: time.Now(),
	}

	if err := s.DB.Create(&address).Error; err != nil {
		return nil, err
	}

	return &pb.AddressResponse{Address: toProto(address)}, nil
}

func (s *AddressServer) GetAddress(ctx context.Context, in *pb.GetAddressRequest) (*pb.AddressResponse, error) {
	var address model.Address
	if err := s.DB.First(&address, in.Id).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "address not found")
	}

	// ✅ Validasi: hanya owner yang boleh ambil data
	if address.OwnerID != uint(in.OwnerId) {
		return nil, status.Errorf(codes.PermissionDenied, "unauthorized: not your address")
	}

	return &pb.AddressResponse{
		Address: &pb.Address{
			Id:        uint32(address.ID),
			Name:      address.Name,
			Desc:      address.Desc,
			OwnerId:   uint32(address.OwnerID),
		},
	}, nil
}

func (s *AddressServer) ListAddresses(ctx context.Context, req *pb.ListAddressRequest) (*pb.ListAddressResponse, error) {
	var addresses []model.Address
	if err := s.DB.Where("owner_id = ?", req.OwnerId).Find(&addresses).Error; err != nil {
		return nil, err
	}

	var protoAddresses []*pb.Address
	for _, addr := range addresses {
		protoAddresses = append(protoAddresses, toProto(addr))
	}

	return &pb.ListAddressResponse{Addresses: protoAddresses}, nil
}

func (s *AddressServer) UpdateAddress(ctx context.Context, req *pb.UpdateAddressRequest) (*pb.AddressResponse, error) {
	var address model.Address
	if err := s.DB.First(&address, req.Id).Error; err != nil {
		return nil, err
	}

	address.Name = req.Name
	address.Desc = req.Desc

	if err := s.DB.Save(&address).Error; err != nil {
		return nil, err
	}

	return &pb.AddressResponse{Address: toProto(address)}, nil
}

func (s *AddressServer) DeleteAddress(ctx context.Context, req *pb.DeleteAddressRequest) (*pb.DeleteAddressResponse, error) {
	if err := s.DB.Delete(&model.Address{}, req.Id).Error; err != nil {
		return nil, err
	}

	return &pb.DeleteAddressResponse{Message: "Address deleted successfully"}, nil
}

func (s *AddressServer) GetAllAddresses(ctx context.Context, _ *emptypb.Empty) (*pb.GetAllAddressesResponse, error) {
    var addresses []model.Address

    if err := s.DB.Find(&addresses).Error; err != nil {
        return nil, err
    }

    var pbAddresses []*pb.Address
    for _, a := range addresses {
        pbAddresses = append(pbAddresses, &pb.Address{
            Id:      uint32(a.ID),
            Name:    a.Name,
            Desc:    a.Desc,
            OwnerId: uint32(a.OwnerID),
        })
    }

    return &pb.GetAllAddressesResponse{Addresses: pbAddresses}, nil
}



func toProto(a model.Address) *pb.Address {
	return &pb.Address{
		Id:         uint32(a.ID),
		Name:       a.Name,
		Desc:       a.Desc,
		OwnerId:    uint32(a.OwnerID),
		CreatedAt:  a.CreatedAt.Format(time.RFC3339),
	}
}
