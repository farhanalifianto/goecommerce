package grpc_server

import (
	"context"
	"encoding/json"
	"time"

	"cart-service/model"
	pb "cart-service/proto/cart"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CartServer struct {
	pb.UnimplementedCartServiceServer
	DB *gorm.DB
}

// CreateCart — tambah produk ke cart
func (s *CartServer) CreateCart(ctx context.Context, req *pb.CreateCartRequest) (*pb.CartResponse, error) {
	var existing model.Cart
	err := s.DB.Where("owner_id = ? AND status = ?", req.OwnerId, "unpaid").First(&existing).Error

	var productsJSON []byte
	newProductsJSON, _ := json.Marshal(req.Products)

	if err == nil {
		// update existing cart
		var existingProducts []*pb.Product
		json.Unmarshal(existing.Products, &existingProducts)

		for _, newProd := range req.Products {
			found := false
			for i, old := range existingProducts {
				if old.ProductId == newProd.ProductId {
					existingProducts[i].Qty += newProd.Qty
					found = true
					break
				}
			}
			if !found {
				existingProducts = append(existingProducts, newProd)
			}
		}

		productsJSON, _ = json.Marshal(existingProducts)
		existing.Products = datatypes.JSON(productsJSON)
		existing.UpdatedAt = time.Now()
		s.DB.Save(&existing)

		return &pb.CartResponse{
			Cart: &pb.Cart{
				Id:        uint32(existing.ID),
				OwnerId:   uint32(existing.OwnerID),
				Status:    existing.Status,
				CreatedAt: existing.CreatedAt.String(),
				UpdatedAt: existing.UpdatedAt.String(),
			},
		}, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	cart := model.Cart{
		OwnerID:   uint(req.OwnerId),
		Products:  datatypes.JSON(newProductsJSON),
		Status:    "unpaid",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.DB.Create(&cart).Error; err != nil {
		return nil, err
	}

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        uint32(cart.ID),
			OwnerId:   uint32(cart.OwnerID),
			Status:    cart.Status,
			CreatedAt: cart.CreatedAt.String(),
			UpdatedAt: cart.UpdatedAt.String(),
		},
	}, nil
}

// GetCart — ambil cart unpaid user
func (s *CartServer) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.CartResponse, error) {
	var cart model.Cart
	if err := s.DB.Where("owner_id = ? AND status = ?", req.OwnerId, "unpaid").First(&cart).Error; err != nil {
		return nil, err
	}

	var products []*pb.Product
	json.Unmarshal(cart.Products, &products)

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        uint32(cart.ID),
			OwnerId:   uint32(cart.OwnerID),
			Products:  products,
			Status:    cart.Status,
			CreatedAt: timestamppb.New(cart.CreatedAt).String(),
			UpdatedAt: timestamppb.New(cart.UpdatedAt).String(),
		},
	}, nil
}

// DeleteProduct — hapus produk dari cart
func (s *CartServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.CartResponse, error) {
	var cart model.Cart
	if err := s.DB.Where("owner_id = ? AND status = ?", req.OwnerId, "unpaid").First(&cart).Error; err != nil {
		return nil, err
	}

	var products []*pb.Product
	json.Unmarshal(cart.Products, &products)

	newProducts := make([]*pb.Product, 0)
	for _, p := range products {
		if p.ProductId != req.ProductId {
			newProducts = append(newProducts, p)
		}
	}

	updatedJSON, _ := json.Marshal(newProducts)
	cart.Products = datatypes.JSON(updatedJSON)
	cart.UpdatedAt = time.Now()
	s.DB.Save(&cart)

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        uint32(cart.ID),
			OwnerId:   uint32(cart.OwnerID),
			Status:    cart.Status,
			CreatedAt: cart.CreatedAt.String(),
			UpdatedAt: cart.UpdatedAt.String(),
		},
	}, nil
}
