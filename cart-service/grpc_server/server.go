package grpc_server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	kafka "cart-service/kafka"
	"cart-service/model"
	pb "cart-service/proto/cart"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CartServer struct {
	pb.UnimplementedCartServiceServer
	DB       *sql.DB
	Producer *kafka.Producer
	Redis    *redis.Client
}

// CREATE
func (s *CartServer) CreateCart(ctx context.Context, req *pb.CreateCartRequest) (*pb.CartResponse, error) {
	// Start tx to avoid concurrent writes
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback()

	// Try to find active cart for this owner and lock it
	var cartID uint32
	var productsRaw []byte
	var createdAt time.Time

	err = tx.QueryRowContext(ctx,
		`SELECT id, products, created_at FROM carts WHERE owner_id=$1 AND status='active' FOR UPDATE`,
		req.OwnerId,
	).Scan(&cartID, &productsRaw, &createdAt)

	// If cart does not exist -> create new and commit
	if err == sql.ErrNoRows {
		// Convert req.Products (proto) -> JSON bytes
		productsBytes, _ := json.Marshal(req.Products)

		insertQuery := `INSERT INTO carts (owner_id, products, status, created_at)
		                VALUES ($1, $2::jsonb, 'active', NOW())
		                RETURNING id, created_at`
		var newID uint32
		err = tx.QueryRowContext(ctx, insertQuery, req.OwnerId, string(productsBytes)).Scan(&newID, &createdAt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create cart: %v", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, status.Errorf(codes.Internal, "tx commit failed: %v", err)
		}

		// Clear redis cache
		cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)
		s.Redis.Del(ctx, cacheKey)

		return &pb.CartResponse{
			Cart: &pb.Cart{
				Id:        newID,
				OwnerId:   req.OwnerId,
				Products:  req.Products,
				Status:    "active",
				CreatedAt: createdAt.Format(time.RFC3339),
			},
		}, nil
	}

	// Other DB error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// If cart exists -> merge incoming products into existing products
	// Unmarshal existing products
	var existing []model.CartProduct
	if len(productsRaw) > 0 {
		if err := json.Unmarshal(productsRaw, &existing); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse existing products json: %v", err)
		}
	} else {
		existing = []model.CartProduct{}
	}

	// Build a map for quick lookup: productID -> index in existing slice
	idxMap := make(map[uint]int, len(existing))
	for i, p := range existing {
		idxMap[p.ID] = i
	}

	// Merge: for each incoming product (proto), add qty if exists else append
	for _, in := range req.Products {
		if in == nil {
			continue
		}
		pid := uint(in.Id)
		if i, ok := idxMap[pid]; ok {
			// increment qty
			existing[i].Qty += uint(in.Qty)
		} else {
			// append new product
			existing = append(existing, model.CartProduct{ID: pid, Qty: uint(in.Qty)})
			idxMap[pid] = len(existing) - 1
		}
	}

	// Convert merged existing -> pb list for response and JSON for DB
	pbProducts := make([]*pb.CartProduct, 0, len(existing))
	for _, p := range existing {
		pbProducts = append(pbProducts, &pb.CartProduct{
			Id:  uint32(p.ID),
			Qty: uint32(p.Qty),
		})
	}
	productsBytes, _ := json.Marshal(pbProducts)

	// Update DB
	updateQuery := `UPDATE carts SET products=$1::jsonb WHERE id=$2 RETURNING created_at`
	err = tx.QueryRowContext(ctx, updateQuery, string(productsBytes), cartID).Scan(&createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update cart: %v", err)
	}

	// commit
	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "tx commit failed: %v", err)
	}

	// Clear redis cache
	cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)
	s.Redis.Del(ctx, cacheKey)

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        cartID,
			OwnerId:   req.OwnerId,
			Products:  pbProducts,
			Status:    "active",
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	}, nil
}



// GET SINGLE
func (s *CartServer) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.CartResponse, error) {
	query := `SELECT id, owner_id, products, status, created_at
              FROM carts WHERE id = $1`

	var c model.Cart
	var productsRaw []byte
	err := s.DB.QueryRowContext(ctx, query, req.Id).
		Scan(&c.ID, &c.OwnerID, &productsRaw, &c.Status, &c.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "cart not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// Unmarshal products JSON into model.CartProduct slice
	if len(productsRaw) > 0 {
		if err := json.Unmarshal(productsRaw, &c.Products); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
		}
	} else {
		c.Products = []model.CartProduct{}
	}

	// Authorization: owner must match
	if c.OwnerID != uint(req.OwnerId) {
		return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
	}

	// Convert model.CartProduct -> pb.CartProduct
	var pbProducts []*pb.CartProduct
	for _, p := range c.Products {
		pbProducts = append(pbProducts, &pb.CartProduct{
			Id:  uint32(p.ID),
			Qty: uint32(p.Qty),
		})
	}

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        uint32(c.ID),
			OwnerId:   uint32(c.OwnerID),
			Products:  pbProducts,
			Status:    c.Status,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *CartServer) ListCarts(ctx context.Context, req *pb.ListCartRequest) (*pb.ListCartResponse, error) {
	cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)

	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var carts []*pb.Cart
		json.Unmarshal([]byte(cached), &carts) // ignore unmarshal error - fallback handled below
		fmt.Println("🔥 Redis HIT")
		return &pb.ListCartResponse{Carts: carts}, nil
	}

	if err != redis.Nil {
		fmt.Println("Redis ERROR (bypass to DB):", err)
	} else {
		fmt.Println("Redis MISS → DB query")
	}

	// 2. Query DB
	query := `SELECT id, owner_id, products, status, created_at
              FROM carts WHERE owner_id = $1`

	rows, err := s.DB.QueryContext(ctx, query, req.OwnerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var carts []*pb.Cart
	for rows.Next() {
		var c model.Cart
		var productsRaw []byte
		if err := rows.Scan(&c.ID, &c.OwnerID, &productsRaw, &c.Status, &c.CreatedAt); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		if len(productsRaw) > 0 {
			if err := json.Unmarshal(productsRaw, &c.Products); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
			}
		} else {
			c.Products = []model.CartProduct{}
		}

		// convert products
		var pbProducts []*pb.CartProduct
		for _, p := range c.Products {
			pbProducts = append(pbProducts, &pb.CartProduct{
				Id:  uint32(p.ID),
				Qty: uint32(p.Qty),
			})
		}

		carts = append(carts, &pb.Cart{
			Id:        uint32(c.ID),
			OwnerId:   uint32(c.OwnerID),
			Products:  pbProducts,
			Status:    c.Status,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}

	// 3. Save to Redis with TTL 5 minutes
	jsonData, _ := json.Marshal(carts)
	if err := s.Redis.Set(ctx, cacheKey, jsonData, 5*time.Minute).Err(); err != nil {
		fmt.Println("⚠ Gagal set Redis:", err)
	}

	return &pb.ListCartResponse{Carts: carts}, nil
}

// UPDATE
func (s *CartServer) UpdateCart(ctx context.Context, req *pb.UpdateCartRequest) (*pb.CartResponse, error) {
	// marshal products
	productsBytes, err := json.Marshal(req.Products)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid products: %v", err)
	}

	query := `UPDATE carts
              SET products=$1::jsonb, status=$2
              WHERE id=$3
              RETURNING id, owner_id, products, status, created_at`

	var c model.Cart
	var productsRaw []byte
	err = s.DB.QueryRowContext(ctx, query, string(productsBytes), req.Status, req.Id).
		Scan(&c.ID, &c.OwnerID, &productsRaw, &c.Status, &c.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "cart not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	// Unmarshal products
	if len(productsRaw) > 0 {
		if err := json.Unmarshal(productsRaw, &c.Products); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
		}
	} else {
		c.Products = []model.CartProduct{}
	}

	// Clear cache
	cacheKey := fmt.Sprintf("carts:%d", c.OwnerID)
	s.Redis.Del(ctx, cacheKey)

	// Publish event (optional)
	// convert products to pb-like structure for event payload
	var eventProducts []map[string]interface{}
	for _, p := range c.Products {
		eventProducts = append(eventProducts, map[string]interface{}{
			"id":  p.ID,
			"qty": p.Qty,
		})
	}
	// event := map[string]interface{}{
	// 	"event_type": "cart_updated",
	// 	"data": map[string]interface{}{
	// 		"id":       c.ID,
	// 		"owner_id": c.OwnerID,
	// 		"products": eventProducts,
	// 		"status":   c.Status,
	// 	},
	// }
	// s.Producer.PublishCartUpdatedEvent(event)

	// convert to pb response
	var pbProducts []*pb.CartProduct
	for _, p := range c.Products {
		pbProducts = append(pbProducts, &pb.CartProduct{
			Id:  uint32(p.ID),
			Qty: uint32(p.Qty),
		})
	}

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        uint32(c.ID),
			OwnerId:   uint32(c.OwnerID),
			Products:  pbProducts,
			Status:    c.Status,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// DELETE
func (s *CartServer) DeleteCart(ctx context.Context, req *pb.DeleteCartRequest) (*pb.DeleteCartResponse, error) {
	// 1. get owner_id of cart
	var ownerID uint32
	err := s.DB.QueryRowContext(ctx, `SELECT owner_id FROM carts WHERE id=$1`, req.Id).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "cart not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// 2. check ownership
	if ownerID != req.OwnerId {
		return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
	}

	// 3. delete
	_, err = s.DB.ExecContext(ctx, `DELETE FROM carts WHERE id=$1`, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete error: %v", err)
	}

	// 4. clear cache
	cacheKey := fmt.Sprintf("carts:%d", ownerID)
	s.Redis.Del(ctx, cacheKey)

	// 5. publish event (optional)
	// event := map[string]interface{}{
	// 	"event_type": "cart_deleted",
	// 	"data": map[string]interface{}{
	// 		"id":       req.Id,
	// 		"owner_id": ownerID,
	// 	},
	// }
	// s.Producer.PublishCartDeletedEvent(event)

	return &pb.DeleteCartResponse{
		Message: "Cart deleted successfully",
	}, nil
}

// GET ALL (no cache)
func (s *CartServer) GetAllCarts(ctx context.Context, _ *emptypb.Empty) (*pb.GetAllCartsResponse, error) {
	query := `SELECT id, owner_id, products, status, created_at FROM carts`

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var carts []*pb.Cart
	for rows.Next() {
		var c model.Cart
		var productsRaw []byte
		if err := rows.Scan(&c.ID, &c.OwnerID, &productsRaw, &c.Status, &c.CreatedAt); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		if len(productsRaw) > 0 {
			if err := json.Unmarshal(productsRaw, &c.Products); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
			}
		} else {
			c.Products = []model.CartProduct{}
		}

		var pbProducts []*pb.CartProduct
		for _, p := range c.Products {
			pbProducts = append(pbProducts, &pb.CartProduct{
				Id:  uint32(p.ID),
				Qty: uint32(p.Qty),
			})
		}

		carts = append(carts, &pb.Cart{
			Id:        uint32(c.ID),
			OwnerId:   uint32(c.OwnerID),
			Products:  pbProducts,
			Status:    c.Status,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.GetAllCartsResponse{Carts: carts}, nil
}

// ==============================
// NEW: AddToCart
// ==============================
func (s *CartServer) AddToCart(ctx context.Context, req *pb.AddToCartRequest) (*pb.CartResponse, error) {
	// Start transaction to avoid races
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback()

	// Find active cart for owner with FOR UPDATE to lock row
	var cartID uint32
	var productsRaw []byte
	var statusStr string
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT id, products, status, created_at FROM carts WHERE owner_id=$1 AND status='active' FOR UPDATE`, req.OwnerId).
		Scan(&cartID, &productsRaw, &statusStr, &createdAt)

	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	var products []model.CartProduct
	if err == sql.ErrNoRows {
		// create new cart
		products = []model.CartProduct{
			{ID: uint(req.ProductId), Qty: uint(req.Qty)},
		}
		pbProducts := []*pb.CartProduct{
			{Id: req.ProductId, Qty: req.Qty},
		}
		productsBytes, _ := json.Marshal(pbProducts)

		insertQuery := `INSERT INTO carts (owner_id, products, status, created_at) VALUES ($1, $2::jsonb, $3, NOW()) RETURNING id, created_at`
		err = tx.QueryRowContext(ctx, insertQuery, req.OwnerId, string(productsBytes), "active").Scan(&cartID, &createdAt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create cart: %v", err)
		}
	} else {
		// existing cart: unmarshal products, update qty or append
		if len(productsRaw) > 0 {
			if err := json.Unmarshal(productsRaw, &products); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
			}
		} else {
			products = []model.CartProduct{}
		}

		found := false
		for i := range products {
			if products[i].ID == uint(req.ProductId) {
				products[i].Qty += uint(req.Qty)
				found = true
				break
			}
		}
		if !found {
			products = append(products, model.CartProduct{ID: uint(req.ProductId), Qty: uint(req.Qty)})
		}

		// marshal back and update
		pbProducts := make([]*pb.CartProduct, 0, len(products))
		for _, p := range products {
			pbProducts = append(pbProducts, &pb.CartProduct{Id: uint32(p.ID), Qty: uint32(p.Qty)})
		}
		productsBytes, _ := json.Marshal(pbProducts)

		updateQuery := `UPDATE carts SET products=$1::jsonb WHERE id=$2 RETURNING created_at`
		err = tx.QueryRowContext(ctx, updateQuery, string(productsBytes), cartID).Scan(&createdAt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update cart: %v", err)
		}
	}

	// commit
	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "tx commit failed: %v", err)
	}

	// Clear redis cache
	cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)
	s.Redis.Del(ctx, cacheKey)

	// Build response cart object
	var pbProducts []*pb.CartProduct
	for _, p := range products {
		pbProducts = append(pbProducts, &pb.CartProduct{
			Id:  uint32(p.ID),
			Qty: uint32(p.Qty),
		})
	}

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        cartID,
			OwnerId:   req.OwnerId,
			Products:  pbProducts,
			Status:    "active",
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *CartServer) UpdateProductQty(ctx context.Context, req *pb.UpdateProductQtyRequest) (*pb.CartResponse, error) {
	// Start tx
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback()

	// Find active cart
	var cartID uint32
	var productsRaw []byte
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT id, products, created_at FROM carts WHERE owner_id=$1 AND status='active' FOR UPDATE`, req.OwnerId).
		Scan(&cartID, &productsRaw, &createdAt)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "active cart not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	var products []model.CartProduct
	if len(productsRaw) > 0 {
		if err := json.Unmarshal(productsRaw, &products); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
		}
	} else {
		products = []model.CartProduct{}
	}

	// update qty or remove if qty == 0
	changed := false
	newList := make([]model.CartProduct, 0, len(products))
	for _, p := range products {
		if p.ID == uint(req.ProductId) {
			if req.Qty == 0 {
				// skip -> remove
				changed = true
				continue
			}
			p.Qty = uint(req.Qty)
			changed = true
		}
		newList = append(newList, p)
	}

	if !changed {
		// product not found and qty > 0 -> append
		if req.Qty > 0 {
			newList = append(newList, model.CartProduct{ID: uint(req.ProductId), Qty: uint(req.Qty)})
			changed = true
		}
	}

	// marshal and update
	pbProducts := make([]*pb.CartProduct, 0, len(newList))
	for _, p := range newList {
		pbProducts = append(pbProducts, &pb.CartProduct{Id: uint32(p.ID), Qty: uint32(p.Qty)})
	}
	productsBytes, _ := json.Marshal(pbProducts)

	updateQuery := `UPDATE carts SET products=$1::jsonb WHERE id=$2 RETURNING created_at`
	err = tx.QueryRowContext(ctx, updateQuery, string(productsBytes), cartID).Scan(&createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update cart: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "tx commit failed: %v", err)
	}

	// Clear redis cache
	cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)
	s.Redis.Del(ctx, cacheKey)

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        cartID,
			OwnerId:   req.OwnerId,
			Products:  pbProducts,
			Status:    "active",
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	}, nil
}

// ==============================
// NEW: RemoveProductFromCart
// ==============================
func (s *CartServer) RemoveProductFromCart(ctx context.Context, req *pb.RemoveProductRequest) (*pb.CartResponse, error) {
	// Reuse UpdateProductQty with qty=0 logic by implementing separately for clarity
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback()

	var cartID uint32
	var productsRaw []byte
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT id, products, created_at FROM carts WHERE owner_id=$1 AND status='active' FOR UPDATE`, req.OwnerId).
		Scan(&cartID, &productsRaw, &createdAt)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "active cart not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	var products []model.CartProduct
	if len(productsRaw) > 0 {
		if err := json.Unmarshal(productsRaw, &products); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
		}
	} else {
		products = []model.CartProduct{}
	}

	newList := []model.CartProduct{}
	removed := false
	for _, p := range products {
		if p.ID == uint(req.ProductId) {
			removed = true
			continue
		}
		newList = append(newList, p)
	}

	if !removed {
		// nothing changed
		return nil, status.Errorf(codes.NotFound, "product not found in cart")
	}

	pbProducts := make([]*pb.CartProduct, 0, len(newList))
	for _, p := range newList {
		pbProducts = append(pbProducts, &pb.CartProduct{Id: uint32(p.ID), Qty: uint32(p.Qty)})
	}
	productsBytes, _ := json.Marshal(pbProducts)

	updateQuery := `UPDATE carts SET products=$1::jsonb WHERE id=$2 RETURNING created_at`
	err = tx.QueryRowContext(ctx, updateQuery, string(productsBytes), cartID).Scan(&createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update cart: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "tx commit failed: %v", err)
	}

	// Clear redis cache
	cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)
	s.Redis.Del(ctx, cacheKey)

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        cartID,
			OwnerId:   req.OwnerId,
			Products:  pbProducts,
			Status:    "active",
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	}, nil
}

// ==============================
// NEW: CheckoutCart
// ==============================
func (s *CartServer) CheckoutCart(ctx context.Context, req *pb.CheckoutRequest) (*pb.CartResponse, error) {
	// Start tx
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin tx: %v", err)
	}
	defer tx.Rollback()

	// Find active cart
	var cartID uint32
	var productsRaw []byte
	var createdAt time.Time
	var currentStatus string
	err = tx.QueryRowContext(ctx, `SELECT id, products, status, created_at FROM carts WHERE owner_id=$1 AND status='active' FOR UPDATE`, req.OwnerId).
		Scan(&cartID, &productsRaw, &currentStatus, &createdAt)
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "active cart not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	// unmarshal products
	var products []model.CartProduct
	if len(productsRaw) > 0 {
		if err := json.Unmarshal(productsRaw, &products); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse products json: %v", err)
		}
	} else {
		products = []model.CartProduct{}
	}

	// Here you would typically:
	// - validate stock with product-service
	// - create transaction record at transaction-service
	// - reduce stock
	// For now we only mark cart as paid

	updateQuery := `UPDATE carts SET status='paid' WHERE id=$1 RETURNING owner_id, products, created_at`
	var ownerID uint32
	var returnedProducts []byte
	err = tx.QueryRowContext(ctx, updateQuery, cartID).Scan(&ownerID, &returnedProducts, &createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to checkout cart: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "tx commit failed: %v", err)
	}

	// Clear redis cache
	cacheKey := fmt.Sprintf("carts:%d", req.OwnerId)
	s.Redis.Del(ctx, cacheKey)

	// publish event (optional)
	// event := map[string]interface{}{
	// 	"event_type": "cart_paid",
	// 	"data": map[string]interface{}{
	// 		"id":       cartID,
	// 		"owner_id": req.OwnerId,
	// 		"products": products,
	// 	},
	// }
	// s.Producer.PublishCartPaidEvent(event)

	// build pb products
	var pbProducts []*pb.CartProduct
	for _, p := range products {
		pbProducts = append(pbProducts, &pb.CartProduct{
			Id:  uint32(p.ID),
			Qty: uint32(p.Qty),
		})
	}

	return &pb.CartResponse{
		Cart: &pb.Cart{
			Id:        cartID,
			OwnerId:   req.OwnerId,
			Products:  pbProducts,
			Status:    "paid",
			CreatedAt: createdAt.Format(time.RFC3339),
		},
	}, nil
}
