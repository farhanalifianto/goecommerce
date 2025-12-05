package grpc_client

import (
	"context"
	"log"
	"time"

	pb "transaction-service/proto/cart"

	"google.golang.org/grpc"
)

type CartClient struct {
	client pb.CartServiceClient
}

func NewCartClient() *CartClient {
	conn, err := grpc.Dial("cart-service:50055", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect to cart-service: %v", err)
	}

	c := pb.NewCartServiceClient(conn)
	return &CartClient{client: c}
}

type CartProductInfo struct {
	Id  uint32
	Qty uint32
}

type CartInfo struct {
	Id        uint32
	OwnerId   uint32
	Products  []CartProductInfo
	Status    string
	CreatedAt string
}

func (cc *CartClient) GetCart(cartID uint32, ownerID uint32) (*CartInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := cc.client.GetCart(ctx, &pb.GetCartRequest{
		Id:       cartID,
		OwnerId:  ownerID,
	})
	if err != nil {
		return nil, err
	}

	cart := res.GetCart()

	// Convert repeated CartProduct → []CartProductInfo
	var products []CartProductInfo
	for _, p := range cart.Products {
		products = append(products, CartProductInfo{
			Id:  p.Id,
			Qty: p.Qty,
		})
	}

	return &CartInfo{
		Id:        cart.Id,
		OwnerId:   cart.OwnerId,
		Products:  products,
		Status:    cart.Status,
		CreatedAt: cart.CreatedAt,
	}, nil
}
