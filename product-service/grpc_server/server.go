package grpc_server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafka "product-service/kafka"
	"product-service/model"
	pb "product-service/proto/product"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"gorm.io/gorm"
)

// ProductServer implements pb.ProductServiceServer
type ProductServer struct {
	pb.UnimplementedProductServiceServer
	DB       *gorm.DB
	Producer *kafka.Producer
	Redis    *redis.Client
}

// helper: format product model -> proto Product
func toProtoProduct(p *model.Product) *pb.Product {
	if p == nil {
		return nil
	}
	var catID uint32 = uint32(p.CategoryID)
	return &pb.Product{
		Id:         uint32(p.ID),
		Name:       p.Name,
		Desc:       p.Desc,
		Price:      uint32(p.Price),
		CategoryId: catID,
		CreatedAt:  p.CreatedAt.Format(time.RFC3339),
	}
}

// ===================== PRODUCT CRUD =====================

// CreateProduct : create + clear relevant cache + publish event
func (s *ProductServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	p := model.Product{
		Name:       req.Name,
		Desc:       req.Desc,
		Price:      uint(req.Price),
		CategoryID: uint(req.CategoryId),
	}

	if err := s.DB.Create(&p).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	// clear list caches
	if s.Redis != nil {
		// remove list caches: all and per-category
		_ = s.Redis.Del(ctx, "products:all").Err()
		_ = s.Redis.Del(ctx, fmt.Sprintf("products:category:%d", p.CategoryID)).Err()
	}

	// publish event
	if s.Producer != nil {
		event := map[string]interface{}{
			"event_type": "product_created",
			"data": map[string]interface{}{
				"id":          p.ID,
				"name":        p.Name,
				"desc":        p.Desc,
				"price":       p.Price,
				"category_id": p.CategoryID,
			},
		}
		s.Producer.PublishProductCreatedEvent(event)
	}

	return &pb.ProductResponse{Product: toProtoProduct(&p)}, nil
}

// GetProduct: return single product by id
func (s *ProductServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	var p model.Product
	if err := s.DB.First(&p, req.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}
	return &pb.ProductResponse{Product: toProtoProduct(&p)}, nil
}

// ListProducts: optional caching (all products)
func (s *ProductServer) ListProducts(ctx context.Context, _ *emptypb.Empty) (*pb.ListProductsResponse, error) {
	// note: proto used google.protobuf.Empty; adjust signature if necessary
	// Cache key "products:all"
	var products []*model.Product

	// Try Redis
	if s.Redis != nil {
		if val, err := s.Redis.Get(ctx, "products:all").Result(); err == nil {
			// Unmarshal into []model.Product
			if err := json.Unmarshal([]byte(val), &products); err == nil {
				// convert to proto
				out := &pb.ListProductsResponse{}
				for _, p := range products {
					out.Products = append(out.Products, toProtoProduct(p))
				}
				return out, nil
			}
			// if unmarshal failed, fallthrough to DB
		}
	}

	// Query DB
	if err := s.DB.Find(&products).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "db query failed: %v", err)
	}

	// Save to Redis
	if s.Redis != nil {
		b, _ := json.Marshal(products)
		_ = s.Redis.Set(ctx, "products:all", b, 5*time.Minute).Err()
	}

	out := &pb.ListProductsResponse{}
	for _, p := range products {
		out.Products = append(out.Products, toProtoProduct(p))
	}
	return out, nil
}

// UpdateProduct: update, clear caches, publish event
func (s *ProductServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	var p model.Product
	if err := s.DB.First(&p, req.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}

	// update fields
	p.Name = req.Name
	p.Desc = req.Desc
	p.Price = uint(req.Price)
	p.CategoryID = uint(req.CategoryId)

	if err := s.DB.Save(&p).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}

	// clear caches
	if s.Redis != nil {
		_ = s.Redis.Del(ctx, "products:all").Err()
		_ = s.Redis.Del(ctx, fmt.Sprintf("products:category:%d", p.CategoryID)).Err()
	}

	// publish event
	if s.Producer != nil {
		event := map[string]interface{}{
			"event_type": "product_updated",
			"data": map[string]interface{}{
				"id":          p.ID,
				"name":        p.Name,
				"desc":        p.Desc,
				"price":       p.Price,
				"category_id": p.CategoryID,
			},
		}
		s.Producer.PublishProductUpdatedEvent(event)
	}

	return &pb.ProductResponse{Product: toProtoProduct(&p)}, nil
}

// DeleteProduct: delete, clear caches, publish event
func (s *ProductServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	var p model.Product
	if err := s.DB.First(&p, req.Id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}

	if err := s.DB.Delete(&model.Product{}, req.Id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "delete failed: %v", err)
	}

	// clear caches
	if s.Redis != nil {
		_ = s.Redis.Del(ctx, "products:all").Err()
		_ = s.Redis.Del(ctx, fmt.Sprintf("products:category:%d", p.CategoryID)).Err()
	}

	// publish event
	if s.Producer != nil {
		event := map[string]interface{}{
			"event_type": "product_deleted",
			"data": map[string]interface{}{
				"id": p.ID,
			},
		}
		s.Producer.PublishProductDeletedEvent(event)
	}

	return &pb.DeleteProductResponse{Message: "Product deleted successfully"}, nil
}

// ===================== CATEGORY =====================

func (s *ProductServer) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CategoryResponse, error) {
	c := model.Category{
		Name: req.Name,
	}
	if err := s.DB.Create(&c).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create category: %v", err)
	}
	// optional: clear any category-list cache if you have one
	return &pb.CategoryResponse{Category: &pb.Category{
		Id:   uint32(c.ID),
		Name: c.Name,
	}}, nil
}

func (s *ProductServer) ListCategories(ctx context.Context, _ *emptypb.Empty) (*pb.ListCategoriesResponse, error) {
	var cats []model.Category
	if err := s.DB.Find(&cats).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}
	out := &pb.ListCategoriesResponse{}
	for _, c := range cats {
		out.Categories = append(out.Categories, &pb.Category{
			Id:   uint32(c.ID),
			Name: c.Name,
		})
	}
	return out, nil
}

// ===================== STOCK =====================

func (s *ProductServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.StockResponse, error) {
	// Try find existing stock row
	var st model.Stock
	err := s.DB.Where("product_id = ?", req.ProductId).First(&st).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// create new
			st = model.Stock{
				ProductID: uint(req.ProductId),
				Quantity:  int(req.Quantity),
			}
			if err := s.DB.Create(&st).Error; err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create stock: %v", err)
			}
		} else {
			return nil, status.Errorf(codes.Internal, "db error: %v", err)
		}
	} else {
		// update existing
		st.Quantity = int(req.Quantity)
		if err := s.DB.Save(&st).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update stock: %v", err)
		}
	}

	// optional: cache invalidation for product details
	if s.Redis != nil {
		_ = s.Redis.Del(ctx, fmt.Sprintf("product:%d", req.ProductId)).Err()
	}

	return &pb.StockResponse{Stock: &pb.Stock{
		Id:        uint32(st.ID),
		ProductId: uint32(st.ProductID),
		Quantity:  int32(st.Quantity),
		UpdatedAt: st.UpdatedAt.Format(time.RFC3339),
	}}, nil
}

func (s *ProductServer) GetStock(ctx context.Context, req *pb.GetStockRequest) (*pb.StockResponse, error) {
	var st model.Stock
	if err := s.DB.Where("product_id = ?", req.ProductId).First(&st).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "stock not found")
		}
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}
	return &pb.StockResponse{Stock: &pb.Stock{
		Id:        uint32(st.ID),
		ProductId: uint32(st.ProductID),
		Quantity:  int32(st.Quantity),
		UpdatedAt: st.UpdatedAt.Format(time.RFC3339),
	}}, nil
}
