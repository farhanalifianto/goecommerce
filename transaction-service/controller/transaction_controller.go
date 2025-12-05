package controller

import (
	"context"
	"strconv"
	"time"

	"transaction-service/grpc_client"
	pb "transaction-service/proto/transaction"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TransactionController struct {
	Client         pb.TransactionServiceClient
	UserClient     *grpc_client.UserClient
	AddressClient  *grpc_client.AddressClient
	CartClient     *grpc_client.CartClient
	ProductClient  *grpc_client.ProductClient
}

func NewTransactionController() *TransactionController {
	// Connect to transaction gRPC service
	conn, err := grpc.Dial("transaction-service:50056", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect transaction gRPC: " + err.Error())
	}

	return &TransactionController{
		Client:         pb.NewTransactionServiceClient(conn),
		UserClient:     grpc_client.NewUserClient(),
		AddressClient:  grpc_client.NewAddressClient(),
		CartClient:     grpc_client.NewCartClient(),
		ProductClient:  grpc_client.NewProductClient(),
	}
}

func (tc *TransactionController) Create(c *fiber.Ctx) error {
    userID := c.Locals("user_id").(uint32)

    var body struct {
        CartID    uint32 `json:"cart_id"`
        AddressID uint32 `json:"address_id"`
    }

    if err := c.BodyParser(&body); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "invalid request body",
        })
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := tc.Client.CreateTransaction(ctx, &pb.CreateTransactionRequest{
        UserId:    userID,
        CartId:    body.CartID,
        AddressId: body.AddressID,
    })

    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.Status(201).JSON(resp.Transaction)
}

func (tc *TransactionController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := tc.Client.GetTransaction(ctx, &pb.GetTransactionRequest{
		Id:     uint32(id),
		UserId: userID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.PermissionDenied {
			return c.Status(403).JSON(fiber.Map{"error": "not the owner"})
		}
		if st.Code() == codes.NotFound {
			return c.Status(404).JSON(fiber.Map{"error": "transaction not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Transaction)
}

func (tc *TransactionController) ListAll(c *fiber.Ctx) error {

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := tc.Client.ListAllTransactions(ctx, &emptypb.Empty{})
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    if resp.Transactions == nil {
        resp.Transactions = []*pb.Transaction{}
    }

    return c.JSON(resp.Transactions)
}
func (tc *TransactionController) ListUser(c *fiber.Ctx) error {
    userID := c.Locals("user_id").(uint32)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := tc.Client.ListUserTransactions(ctx, &pb.ListTransactionRequest{
        UserId: userID,
    })
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    if resp.Transactions == nil {
        resp.Transactions = []*pb.Transaction{}
    }

    return c.JSON(resp.Transactions)
}


func (tc *TransactionController) Cancel(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := tc.Client.CancelTransaction(ctx, &pb.CancelTransactionRequest{
		Id:     uint32(id),
		UserId: userID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.PermissionDenied {
			return c.Status(403).JSON(fiber.Map{"error": "not the owner"})
		}
		if st.Code() == codes.FailedPrecondition {
			return c.Status(400).JSON(fiber.Map{"error": st.Message()})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}
