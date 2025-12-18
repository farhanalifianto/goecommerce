package grpc_server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	grpc_client "transaction-service/grpc_client"
	kafka "transaction-service/kafka"
	"transaction-service/model"
	pb "transaction-service/proto/transaction"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TransactionServer struct {
	pb.UnimplementedTransactionServiceServer
	DB       *sql.DB
	Producer *kafka.Producer
	Redis    *redis.Client

	UserClient    *grpc_client.UserClient
	CartClient    *grpc_client.CartClient
	AddressClient *grpc_client.AddressClient
	ProductClient *grpc_client.ProductClient
}

func NewTransactionServer(db *sql.DB, prod *kafka.Producer, rdb *redis.Client) *TransactionServer {
	return &TransactionServer{
		DB:            db,
		Producer:      prod,
		Redis:         rdb,
		UserClient:    grpc_client.NewUserClient(),
		CartClient:    grpc_client.NewCartClient(),
		AddressClient: grpc_client.NewAddressClient(),
		ProductClient: grpc_client.NewProductClient(),
	}
}


func (s *TransactionServer) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.TransactionResponse, error) {

	// Address snapshot
	addrInfo, err := s.AddressClient.GetAddress(req.AddressId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "address not found: %v", err)
	}

	addrSnap := model.AddressSnapshot{
		AddressID: addrInfo.Id,
		Name:      addrInfo.Name,
		Desc:      addrInfo.Desc,
	}

	addrJSON, _ := json.Marshal(addrSnap)

	// Cart snapshot
	cartInfo, err := s.CartClient.GetCart(req.CartId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "cart not found: %v", err)
	}
	if cartInfo.Status != "active" {
		return nil, status.Errorf(codes.FailedPrecondition, "cart not active")
	}

	// Product snapshots
	var productSnaps []model.ProductSnapshot
	var total int64 = 0

	for _, item := range cartInfo.Products {
		pInfo, err := s.ProductClient.GetProduct(item.Id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "product %d not found: %v", item.Id, err)
		}

		sub := int64(pInfo.Price) * int64(item.Qty)
		total += sub

		productSnaps = append(productSnaps, model.ProductSnapshot{
			ProductID:  pInfo.Id,
			Name:       pInfo.Name,
			Price:      int64(pInfo.Price),
			Qty:        item.Qty,
			Subtotal:   sub,
			CategoryID: pInfo.CategoryId,
		})
	}

	productJSON, _ := json.Marshal(productSnaps)

	// Insert transaction
	insertQ := `
        INSERT INTO transactions
        (user_id, cart_id, address_snapshot, product_snapshot, total_amount, status, created_at)
        VALUES ($1,$2,$3::jsonb,$4::jsonb,$5,'pending',NOW())
        RETURNING id, created_at
    `

	var id uint32
	var createdAt time.Time

	err = s.DB.QueryRowContext(
		ctx,
		insertQ,
		req.UserId,
		req.CartId,
		string(addrJSON),
		string(productJSON),
		total,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}

	// clear user's cache
	s.Redis.Del(ctx, fmt.Sprintf("transactions:%d", req.UserId))
	// also clear admin/all cache
	s.Redis.Del(ctx, "transactions:all")

	event := map[string]interface{}{
    "event_type": "cart.checked_out",
    "data": map[string]interface{}{
        "cart_id":        req.CartId,
        "user_id":        req.UserId,
        "transaction_id": id,
        "total_amount":   total,
        "checked_out_at": time.Now().Format(time.RFC3339),
		},
	}

	s.Producer.PublishCartCheckedOutEvent(event)

	return &pb.TransactionResponse{
		Transaction: &pb.Transaction{
			Id:          id,
			UserId:      req.UserId,
			CartId:      req.CartId,
			Address:     toProtoAddress(addrSnap),
			Products:    toProtoProductSnapList(productSnaps),
			TotalAmount: total,
			Status:      "pending",
			CreatedAt:   createdAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *TransactionServer) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.TransactionResponse, error) {

	q := `
        SELECT id, user_id, cart_id, address_snapshot, product_snapshot,
               total_amount, status, created_at, paid_at
        FROM transactions WHERE id=$1
    `

	var (
		id        uint32
		userID    uint32
		cartID    uint32
		addrRaw   []byte
		prodRaw   []byte
		total     int64
		statusStr string
		createdAt time.Time
		paidAt    sql.NullTime
	)

	err := s.DB.QueryRowContext(ctx, q, req.Id).Scan(
		&id, &userID, &cartID, &addrRaw, &prodRaw, &total,
		&statusStr, &createdAt, &paidAt,
	)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if userID != req.UserId {
		return nil, status.Errorf(codes.PermissionDenied, "unauthorized")
	}

	var addrSnap model.AddressSnapshot
	var prodSnaps []model.ProductSnapshot

	_ = json.Unmarshal(addrRaw, &addrSnap)
	_ = json.Unmarshal(prodRaw, &prodSnaps)

	var paidAtStr string
	if paidAt.Valid {
		paidAtStr = paidAt.Time.Format(time.RFC3339)
	}

	return &pb.TransactionResponse{
		Transaction: &pb.Transaction{
			Id:          id,
			UserId:      userID,
			CartId:      cartID,
			Address:     toProtoAddress(addrSnap),
			Products:    toProtoProductSnapList(prodSnaps),
			TotalAmount: total,
			Status:      statusStr,
			CreatedAt:   createdAt.Format(time.RFC3339),
			PaidAt:      paidAtStr,
		},
	}, nil
}

func (s *TransactionServer) ListUserTransactions(ctx context.Context, req *pb.ListTransactionRequest) (*pb.ListTransactionResponse, error) {

    q := `
        SELECT id, user_id, cart_id, address_snapshot, product_snapshot,
               total_amount, status, created_at, paid_at
        FROM transactions
        WHERE user_id=$1
        ORDER BY created_at DESC
    `

    rows, err := s.DB.QueryContext(ctx, q, req.UserId)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query error: %v", err)
    }
    defer rows.Close()

    var list []*pb.Transaction

    for rows.Next() {
        var (
            id uint32
            uid uint32
            cartID uint32
            addrRaw []byte
            prodRaw []byte
            total int64
            statusStr string
            createdAt time.Time
            paidAt sql.NullTime
        )

        if err := rows.Scan(&id, &uid, &cartID, &addrRaw, &prodRaw,
            &total, &statusStr, &createdAt, &paidAt); err != nil {
            return nil, status.Errorf(codes.Internal, "scan error: %v", err)
        }

        var addrSnap model.AddressSnapshot
        var prodSnaps []model.ProductSnapshot
        _ = json.Unmarshal(addrRaw, &addrSnap)
        _ = json.Unmarshal(prodRaw, &prodSnaps)

        paidAtStr := ""
        if paidAt.Valid {
            paidAtStr = paidAt.Time.Format(time.RFC3339)
        }

        list = append(list, &pb.Transaction{
            Id:          id,
            UserId:      uid,
            CartId:      cartID,
            Address:     toProtoAddress(addrSnap),
            Products:    toProtoProductSnapList(prodSnaps),
            TotalAmount: total,
            Status:      statusStr,
            CreatedAt:   createdAt.Format(time.RFC3339),
            PaidAt:      paidAtStr,
        })
    }

    return &pb.ListTransactionResponse{Transactions: list}, nil
}


func (s *TransactionServer) ListAllTransactions(ctx context.Context, _ *emptypb.Empty) (*pb.ListTransactionResponse, error) {

    q := `
        SELECT id, user_id, cart_id, address_snapshot, product_snapshot,
               total_amount, status, created_at, paid_at
        FROM transactions
        ORDER BY created_at DESC
    `

    rows, err := s.DB.QueryContext(ctx, q)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query error: %v", err)
    }
    defer rows.Close()

    var list []*pb.Transaction

    for rows.Next() {
        var (
            id uint32
            uid uint32
            cartID uint32
            addrRaw []byte
            prodRaw []byte
            total int64
            statusStr string
            createdAt time.Time
            paidAt sql.NullTime
        )

        if err := rows.Scan(&id, &uid, &cartID, &addrRaw, &prodRaw,
            &total, &statusStr, &createdAt, &paidAt); err != nil {
            return nil, status.Errorf(codes.Internal, "scan error: %v", err)
        }

        var addrSnap model.AddressSnapshot
        var prodSnaps []model.ProductSnapshot
        _ = json.Unmarshal(addrRaw, &addrSnap)
        _ = json.Unmarshal(prodRaw, &prodSnaps)

        paidAtStr := ""
        if paidAt.Valid {
            paidAtStr = paidAt.Time.Format(time.RFC3339)
        }

        list = append(list, &pb.Transaction{
            Id:          id,
            UserId:      uid,
            CartId:      cartID,
            Address:     toProtoAddress(addrSnap),
            Products:    toProtoProductSnapList(prodSnaps),
            TotalAmount: total,
            Status:      statusStr,
            CreatedAt:   createdAt.Format(time.RFC3339),
            PaidAt:      paidAtStr,
        })
    }

    return &pb.ListTransactionResponse{Transactions: list}, nil
}

func (s *TransactionServer) MarkAsPaid(ctx context.Context, req *pb.MarkAsPaidRequest) (*pb.TransactionResponse, error) {

	q := `
        UPDATE transactions
        SET status='paid', paid_at=NOW()
        WHERE id=$1
        RETURNING id, user_id, cart_id, address_snapshot, product_snapshot,
                  total_amount, status, created_at, paid_at
    `

	var (
		id        uint32
		userID    uint32
		cartID    uint32
		addrRaw   []byte
		prodRaw   []byte
		total     int64
		statusStr string
		createdAt time.Time
		paidAt    sql.NullTime
	)

	err := s.DB.QueryRowContext(ctx, q, req.Id).Scan(
		&id, &userID, &cartID, &addrRaw, &prodRaw, &total,
		&statusStr, &createdAt, &paidAt,
	)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	// clear caches
	s.Redis.Del(ctx, fmt.Sprintf("transactions:%d", userID))
	s.Redis.Del(ctx, "transactions:all")

	var addrSnap model.AddressSnapshot
	var prodSnaps []model.ProductSnapshot

	_ = json.Unmarshal(addrRaw, &addrSnap)
	_ = json.Unmarshal(prodRaw, &prodSnaps)

	var paidAtStr string
	if paidAt.Valid {
		paidAtStr = paidAt.Time.Format(time.RFC3339)
	}

	return &pb.TransactionResponse{
		Transaction: &pb.Transaction{
			Id:          id,
			UserId:      userID,
			CartId:      cartID,
			Address:     toProtoAddress(addrSnap),
			Products:    toProtoProductSnapList(prodSnaps),
			TotalAmount: total,
			Status:      statusStr,
			CreatedAt:   createdAt.Format(time.RFC3339),
			PaidAt:      paidAtStr,
		},
	}, nil
}


func (s *TransactionServer) CancelTransaction(ctx context.Context, req *pb.CancelTransactionRequest) (*pb.CancelTransactionResponse, error) {

	var (
		userID    uint32
		trxStatus string
	)

	err := s.DB.QueryRowContext(
		ctx,
		`SELECT user_id, status FROM transactions WHERE id=$1`,
		req.Id,
	).Scan(&userID, &trxStatus)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	if userID != req.UserId {
		return nil, status.Errorf(codes.PermissionDenied, "you cannot cancel this transaction")
	}

	if trxStatus == "paid" {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot cancel paid transaction")
	}

	_, err = s.DB.ExecContext(ctx, `UPDATE transactions SET status='cancelled' WHERE id=$1`, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel transaction: %v", err)
	}

	// clear caches
	s.Redis.Del(ctx, fmt.Sprintf("transactions:%d", userID))
	s.Redis.Del(ctx, "transactions:all")

	return &pb.CancelTransactionResponse{
		Message: "transaction successfully cancelled",
	}, nil
}


func toProtoAddress(a model.AddressSnapshot) *pb.AddressSnapshot {
	return &pb.AddressSnapshot{
		AddressId: a.AddressID,
		Name:      a.Name,
		Desc:      a.Desc,
	}
}

func toProtoProductSnapList(list []model.ProductSnapshot) []*pb.ProductSnapshot {
	out := make([]*pb.ProductSnapshot, 0, len(list))
	for _, p := range list {
		out = append(out, &pb.ProductSnapshot{
			ProductId:  p.ProductID,
			Name:       p.Name,
			Price:      p.Price,
			Qty:        p.Qty,
			Subtotal:   p.Subtotal,
			CategoryId: p.CategoryID,
		})
	}
	return out
}
