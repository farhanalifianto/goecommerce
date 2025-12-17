package grpc_client

import (
	"context"
	"log"
	"time"

	pb "payment-service/proto/transaction" // hasil generate proto transaction

	"google.golang.org/grpc"
)

type TransactionClient struct {
	client pb.TransactionServiceClient
}

func NewTransactionClient() *TransactionClient {
	conn, err := grpc.Dial("transaction-service:50056", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect to transaction-service: %v", err)
	}

	c := pb.NewTransactionServiceClient(conn)
	return &TransactionClient{client: c}
}

// ---------- DTO (biar gak expose proto ke service) ----------

type TransactionInfo struct {
	Id          uint32
	UserId      uint32
	TotalAmount int64
	Status      string
	CreatedAt   string
}

// ---------- METHODS ----------

// ambil 1 transaction (dipakai payment buat ambil amount)
func (tc *TransactionClient) GetTransaction(
	transactionID uint32,
	userID uint32,
) (*TransactionInfo, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	res, err := tc.client.GetTransaction(ctx, &pb.GetTransactionRequest{
		Id:     transactionID,
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	t := res.Transaction

	return &TransactionInfo{
		Id:          t.Id,
		UserId:      t.UserId,
		TotalAmount: t.TotalAmount,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
	}, nil
}
