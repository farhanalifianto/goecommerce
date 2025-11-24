package grpc_server

import (
	"context"
	"database/sql"
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
)

type ProductServer struct {
	pb.UnimplementedProductServiceServer
	DB       *sql.DB
	Producer *kafka.Producer
	Redis    *redis.Client
}

// ====================== HELPER ======================

func toProtoProduct(p *model.Product) *pb.Product {
	if p == nil {
		return nil
	}
	return &pb.Product{
		Id:         uint32(p.ID),
		Name:       p.Name,
		Desc:       p.Desc,
		Price:      uint32(p.Price),
		CategoryId: uint32(p.CategoryID),
		CreatedAt:  p.CreatedAt.Format(time.RFC3339),
	}
}

// product

// CREATE
func (s *ProductServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	query := `
	INSERT INTO products (name, "desc", price, category_id, created_at)
	VALUES ($1, $2, $3, $4, NOW())
	RETURNING id, name, "desc", price, category_id, created_at
	`

	var p model.Product
	err := s.DB.QueryRowContext(
		ctx, query,
		req.Name, req.Desc, req.Price, req.CategoryId,
	).Scan(&p.ID, &p.Name, &p.Desc, &p.Price, &p.CategoryID, &p.CreatedAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert: %v", err)
	}

	// clear cache
	s.Redis.Del(ctx, "products:all")
	s.Redis.Del(ctx, fmt.Sprintf("products:category:%d", p.CategoryID))

	// publish event
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

	return &pb.ProductResponse{Product: toProtoProduct(&p)}, nil
}

// GET SINGLE
func (s *ProductServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	query := `
	SELECT id, name, "desc", price, category_id, created_at
	FROM products WHERE id = $1
	`

	var p model.Product
	err := s.DB.QueryRowContext(ctx, query, req.Id).
		Scan(&p.ID, &p.Name, &p.Desc, &p.Price, &p.CategoryID, &p.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	return &pb.ProductResponse{Product: toProtoProduct(&p)}, nil
}

// LIST (with Redis cache)
func (s *ProductServer) ListProducts(ctx context.Context, _ *emptypb.Empty) (*pb.ListProductsResponse, error) {
	cacheKey := "products:all"

	// 1. Try Redis
	if cached, err := s.Redis.Get(ctx, cacheKey).Result(); err == nil {
		fmt.Println("🔥 Product Cache HIT")

		var products []*model.Product
		json.Unmarshal([]byte(cached), &products)

		resp := &pb.ListProductsResponse{}
		for _, p := range products {
			resp.Products = append(resp.Products, toProtoProduct(p))
		}
		return resp, nil
	}

	// 2. Query DB
	query := `
	SELECT id, name, "desc", price, category_id, created_at
	FROM products
	`

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Desc, &p.Price, &p.CategoryID, &p.CreatedAt); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		products = append(products, &p)
	}

	// save to Redis
	b, _ := json.Marshal(products)
	s.Redis.Set(ctx, cacheKey, b, 5*time.Minute)

	resp := &pb.ListProductsResponse{}
	for _, p := range products {
		resp.Products = append(resp.Products, toProtoProduct(p))
	}
	return resp, nil
}

// UPDATE
func (s *ProductServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	query := `
	UPDATE products 
	SET name=$1, "desc"=$2, price=$3, category_id=$4
	WHERE id=$5
	RETURNING id, name, "desc", price, category_id, created_at
	`

	var p model.Product
	err := s.DB.QueryRowContext(
		ctx, query,
		req.Name, req.Desc, req.Price, req.CategoryId, req.Id,
	).Scan(&p.ID, &p.Name, &p.Desc, &p.Price, &p.CategoryID, &p.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	// clear cache
	s.Redis.Del(ctx, "products:all")
	s.Redis.Del(ctx, fmt.Sprintf("products:category:%d", p.CategoryID))

	// publish event
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

	return &pb.ProductResponse{Product: toProtoProduct(&p)}, nil
}

// DELETE
func (s *ProductServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	// 1. Check existing product
	var categoryID uint32
	err := s.DB.QueryRowContext(
		ctx, `SELECT category_id FROM products WHERE id=$1`, req.Id,
	).Scan(&categoryID)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// 2. Delete
	_, err = s.DB.ExecContext(ctx, `DELETE FROM products WHERE id=$1`, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete error: %v", err)
	}

	// 3. Clear cache
	s.Redis.Del(ctx, "products:all")
	s.Redis.Del(ctx, fmt.Sprintf("products:category:%d", categoryID))

	// 4. Publish event
	event := map[string]interface{}{
		"event_type": "product_deleted",
		"data": map[string]interface{}{
			"id": req.Id,
		},
	}
	s.Producer.PublishProductDeletedEvent(event)

	return &pb.DeleteProductResponse{Message: "Product deleted successfully"}, nil
}

// ====================== CATEGORY ======================

func (s *ProductServer) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CategoryResponse, error) {
	query := `
	INSERT INTO categories (name) VALUES ($1)
	RETURNING id, name
	`

	var c model.Category
	err := s.DB.QueryRowContext(ctx, query, req.Name).
		Scan(&c.ID, &c.Name)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert: %v", err)
	}

	return &pb.CategoryResponse{
		Category: &pb.Category{
			Id:   uint32(c.ID),
			Name: c.Name,
		},
	}, nil
}

func (s *ProductServer) ListCategories(ctx context.Context, _ *emptypb.Empty) (*pb.ListCategoriesResponse, error) {
	query := `
	SELECT id, name FROM categories
	`

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	resp := &pb.ListCategoriesResponse{}
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		resp.Categories = append(resp.Categories, &pb.Category{
			Id:   uint32(c.ID),
			Name: c.Name,
		})
	}

	return resp, nil
}
func (s *ProductServer) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.CategoryResponse, error) {
	query := `
	UPDATE categories 
	SET name = $1
	WHERE id = $2
	RETURNING id, name
	`

	var c model.Category
	err := s.DB.QueryRowContext(
		ctx, query,
		req.Name, req.Id,
	).Scan(&c.ID, &c.Name)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "category not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	return &pb.CategoryResponse{
		Category: &pb.Category{
			Id:   uint32(c.ID),
			Name: c.Name,
		},
	}, nil
}
func (s *ProductServer) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*pb.DeleteCategoryResponse, error) {
	// 1. Check if category exists
	var exists bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM categories WHERE id=$1)`,
		req.Id,
	).Scan(&exists)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "category not found")
	}

	// 2. Check if category used by products
	var used bool
	err = s.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM products WHERE category_id=$1)`,
		req.Id,
	).Scan(&used)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}
	if used {
		return nil, status.Errorf(codes.FailedPrecondition, "category in use by products")
	}

	// 3. Delete
	_, err = s.DB.ExecContext(ctx,
		`DELETE FROM categories WHERE id=$1`,
		req.Id,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete error: %v", err)
	}

	return &pb.DeleteCategoryResponse{
		Message: "Category deleted successfully",
	}, nil
}


// ====================== STOCK ======================

func (s *ProductServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.StockResponse, error) {
	var exists bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM stocks WHERE product_id=$1)`,
		req.ProductId,
	).Scan(&exists)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}

	var st model.Stock

	if !exists {
		// INSERT
		query := `
		INSERT INTO stocks (product_id, quantity, updated_at)
		VALUES ($1, $2, NOW())
		RETURNING id, product_id, quantity, updated_at
		`
		err = s.DB.QueryRowContext(ctx, query, req.ProductId, req.Quantity).
			Scan(&st.ID, &st.ProductID, &st.Quantity, &st.UpdatedAt)
	} else {
		// UPDATE
		query := `
		UPDATE stocks SET quantity=$1, updated_at=NOW()
		WHERE product_id=$2
		RETURNING id, product_id, quantity, updated_at
		`
		err = s.DB.QueryRowContext(ctx, query, req.Quantity, req.ProductId).
			Scan(&st.ID, &st.ProductID, &st.Quantity, &st.UpdatedAt)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "stock update error: %v", err)
	}

	return &pb.StockResponse{
		Stock: &pb.Stock{
			Id:        uint32(st.ID),
			ProductId: uint32(st.ProductID),
			Quantity:  int32(st.Quantity),
			UpdatedAt: st.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *ProductServer) GetStock(ctx context.Context, req *pb.GetStockRequest) (*pb.StockResponse, error) {
	query := `
	SELECT id, product_id, quantity, updated_at
	FROM stocks WHERE product_id=$1
	`

	var st model.Stock
	err := s.DB.QueryRowContext(ctx, query, req.ProductId).
		Scan(&st.ID, &st.ProductID, &st.Quantity, &st.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "stock not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}

	return &pb.StockResponse{
		Stock: &pb.Stock{
			Id:        uint32(st.ID),
			ProductId: uint32(st.ProductID),
			Quantity:  int32(st.Quantity),
			UpdatedAt: st.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}
