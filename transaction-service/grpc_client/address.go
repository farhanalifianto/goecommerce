package grpc_client

import (
	"context"
	"log"
	"time"

	pb "transaction-service/proto/address"

	"google.golang.org/grpc"
)

type AddressClient struct {
	client pb.AddressServiceClient
}

func NewAddressClient() *AddressClient {
	conn, err := grpc.Dial("address-service:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect to address-service: %v", err)
	}

	c := pb.NewAddressServiceClient(conn)
	return &AddressClient{client: c}
}

type AddressInfo struct {
	Id        uint32
	Name      string
	Desc      string
	OwnerId   uint32
	CreatedAt string
}

func (ac *AddressClient) GetAddress(id uint32, ownerID uint32) (*AddressInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	res, err := ac.client.GetAddress(ctx, &pb.GetAddressRequest{
		Id:       id,
		OwnerId:  ownerID,
	})
	if err != nil {
		return nil, err
	}

	addr := res.GetAddress()

	return &AddressInfo{
		Id:        addr.Id,
		Name:      addr.Name,
		Desc:      addr.Desc,
		OwnerId:   addr.OwnerId,
		CreatedAt: addr.CreatedAt,
	}, nil
}
