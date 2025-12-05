package grpc_client

import (
	"context"
	"log"
	"time"

	pb "transaction-service/proto/product"

	"google.golang.org/grpc"
)

type ProductClient struct {
	client pb.ProductServiceClient
}

func NewProductClient() *ProductClient {
	conn, err := grpc.Dial("product-service:50054", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect to product-service: %v", err)
	}

	c := pb.NewProductServiceClient(conn)
	return &ProductClient{client: c}
}

type ProductInfo struct {
	Id         uint32
	Name       string
	Desc       string
	Price      uint32
	CategoryId uint32
	CreatedAt  string
}

func (pc *ProductClient) GetProduct(id uint32) (*ProductInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := pc.client.GetProduct(ctx, &pb.GetProductRequest{Id: id})
	if err != nil {
		return nil, err
	}

	product := res.GetProduct()
	return &ProductInfo{
		Id:         product.Id,
		Name:       product.Name,
		Desc:       product.Desc,
		Price:      product.Price,
		CategoryId: product.CategoryId,
		CreatedAt:  product.CreatedAt,
	}, nil
}
