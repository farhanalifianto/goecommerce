package grpc_server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	kafka "payment-service/kafka"
	"payment-service/model"
	pb "payment-service/proto/payment"

	grpc_client "payment-service/grpc_client"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	DB       *sql.DB
	Producer *kafka.Producer
	Redis    *redis.Client
    TransactionClient *grpc_client.TransactionClient
}
func NewPaymentServer(db *sql.DB, rdb *redis.Client, prod *kafka.Producer) *PaymentServer {
	return &PaymentServer{
		DB:                db,
		Redis:             rdb,
		Producer:          prod,
		TransactionClient: grpc_client.NewTransactionClient(),
	}
}

///create payment
func (s *PaymentServer) CreatePayment(ctx context.Context,req *pb.CreatePaymentRequest,) (*pb.PaymentResponse, error) {

	// 1. Ambil data transaction dari TransactionService
	tx, err := s.TransactionClient.GetTransaction(
		req.TransactionId,
		req.UserId,
	)
	if err != nil {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"failed to get transaction: %v",
			err,
		)
	}

	// 2. Validasi status transaction
	if tx.Status != "pending" {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"transaction already processed",
		)
	}

	amount := tx.TotalAmount

	// 3. Insert payment (amount dari transaction)
	query := `
		INSERT INTO payments
		(transaction_id, user_id, amount, status, method, created_at)
		VALUES ($1,$2,$3,'pending','manual',NOW())
		RETURNING id, created_at
	`

	var (
		id        uint32
		createdAt time.Time
	)

	err = s.DB.QueryRowContext(
		ctx,
		query,
		req.TransactionId,
		req.UserId,
		amount,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"failed to create payment: %v",
			err,
		)
	}

	// 4. Clear cache
	s.Redis.Del(ctx, fmt.Sprintf("payments:%d", req.UserId))
	s.Redis.Del(ctx, "payments:all")

	return &pb.PaymentResponse{
		Payment: &pb.Payment{
			Id:            id,
			TransactionId: req.TransactionId,
			UserId:        req.UserId,
			Amount:        amount,
			Status:        "pending",
			Method:        "manual",
			CreatedAt:     createdAt.Format(time.RFC3339),
		},
	}, nil
}

// get payment by id
func (s *PaymentServer) GetPayment(ctx context.Context,req *pb.GetPaymentRequest,) (*pb.PaymentResponse, error) {

	query := `
		SELECT id, transaction_id, user_id, amount, status, method, created_at, paid_at
		FROM payments WHERE id=$1
	`

	var (
		p model.Payment
		paidAt sql.NullTime
	)

	err := s.DB.QueryRowContext(ctx, query, req.Id).
		Scan(
			&p.ID,
			&p.TransactionID,
			&p.UserID,
			&p.Amount,
			&p.Status,
			&p.Method,
			&p.CreatedAt,
			&paidAt,
		)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "payment not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}


	paidAtStr := ""
	if paidAt.Valid {
		paidAtStr = paidAt.Time.Format(time.RFC3339)
	}

	return &pb.PaymentResponse{
		Payment: &pb.Payment{
			Id:            uint32(p.ID),
			TransactionId: uint32(p.TransactionID),
			UserId:        uint32(p.UserID),
			Amount:        p.Amount,
			Status:        p.Status,
			Method:        p.Method,
			CreatedAt:     p.CreatedAt.Format(time.RFC3339),
			PaidAt:        paidAtStr,
		},
	}, nil
}

/// list payment by user
func (s *PaymentServer) ListUserPayments(ctx context.Context,req *pb.ListPaymentRequest,) (*pb.ListPaymentResponse, error) {

	cacheKey := fmt.Sprintf("payments:%d", req.UserId)

	cached, err := s.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var list []*pb.Payment
		_ = json.Unmarshal([]byte(cached), &list)
		return &pb.ListPaymentResponse{Payments: list}, nil
	}

	query := `
		SELECT id, transaction_id, user_id, amount, status, method, created_at, paid_at
		FROM payments WHERE user_id=$1
		ORDER BY created_at DESC
	`

	rows, err := s.DB.QueryContext(ctx, query, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var list []*pb.Payment

	for rows.Next() {
		var (
			p model.Payment
			paidAt sql.NullTime
		)

		rows.Scan(
			&p.ID,
			&p.TransactionID,
			&p.UserID,
			&p.Amount,
			&p.Status,
			&p.Method,
			&p.CreatedAt,
			&paidAt,
		)

		paidAtStr := ""
		if paidAt.Valid {
			paidAtStr = paidAt.Time.Format(time.RFC3339)
		}

		list = append(list, &pb.Payment{
			Id:            uint32(p.ID),
			TransactionId: uint32(p.TransactionID),
			UserId:        uint32(p.UserID),
			Amount:        p.Amount,
			Status:        p.Status,
			Method:        p.Method,
			CreatedAt:     p.CreatedAt.Format(time.RFC3339),
			PaidAt:        paidAtStr,
		})
	}

	js, _ := json.Marshal(list)
	s.Redis.Set(ctx, cacheKey, js, 5*time.Minute)

	return &pb.ListPaymentResponse{Payments: list}, nil
}

//// pay payment
func (s *PaymentServer) PayPayment(ctx context.Context,req *pb.PayPaymentRequest,) (*pb.PaymentResponse, error) {

	query := `
		UPDATE payments
		SET status='paid', paid_at=NOW()
		WHERE id=$1 AND user_id=$2
		RETURNING transaction_id, amount, paid_at
	`

	var (
		transactionID uint32
		amount        int64
		paidAt        time.Time
	)

	err := s.DB.QueryRowContext(
		ctx,
		query,
		req.Id,
		req.UserId,
	).Scan(&transactionID, &amount, &paidAt)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "payment not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "pay error: %v", err)
	}

	// clear cache
	s.Redis.Del(ctx, fmt.Sprintf("payments:%d", req.UserId))
	s.Redis.Del(ctx, "payments:all")

	event := map[string]interface{}{
	"event_type": "payment.paid",
	"data": map[string]interface{}{
		"payment_id":     req.Id,
		"transaction_id": transactionID,
		"user_id":        req.UserId,
		"amount":         amount,
		"paid_at":        paidAt.Format(time.RFC3339),
	},
	}

	s.Producer.PublishPaymentPaidEvent(event)
		return &pb.PaymentResponse{
		Payment: &pb.Payment{
			Id:        req.Id,
			Status:    "paid",
			PaidAt:   paidAt.Format(time.RFC3339),
		},
	}, nil
}

/// list all payment
func (s *PaymentServer) ListAllPayments(ctx context.Context,_ *emptypb.Empty,
) (*pb.ListPaymentResponse, error) {

	query := `
		SELECT id, transaction_id, user_id, amount, status, method, created_at, paid_at
		FROM payments
		ORDER BY created_at DESC
	`

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var list []*pb.Payment

	for rows.Next() {
		var (
			p model.Payment
			paidAt sql.NullTime
		)

		rows.Scan(
			&p.ID,
			&p.TransactionID,
			&p.UserID,
			&p.Amount,
			&p.Status,
			&p.Method,
			&p.CreatedAt,
			&paidAt,
		)

		paidAtStr := ""
		if paidAt.Valid {
			paidAtStr = paidAt.Time.Format(time.RFC3339)
		}

		list = append(list, &pb.Payment{
			Id:            uint32(p.ID),
			TransactionId: uint32(p.TransactionID),
			UserId:        uint32(p.UserID),
			Amount:        p.Amount,
			Status:        p.Status,
			Method:        p.Method,
			CreatedAt:     p.CreatedAt.Format(time.RFC3339),
			PaidAt:        paidAtStr,
		})
	}

	return &pb.ListPaymentResponse{Payments: list}, nil
}
