package controller

import (
	"context"
	"payment-service/grpc_client"
	pb "payment-service/proto/payment"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PaymentController struct {
	Client       pb.PaymentServiceClient
	UserClient   *grpc_client.UserClient
}

func NewPaymentController() *PaymentController {
	conn, err := grpc.Dial("localhost:50057", grpc.WithInsecure()) // port payment-service
	if err != nil {
		panic("failed to connect to payment gRPC: " + err.Error())
	}

	return &PaymentController{
		Client:     pb.NewPaymentServiceClient(conn),
		UserClient: grpc_client.NewUserClient(),
	}
}


func (pc *PaymentController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint32)

	var body struct {
		TransactionID uint32 `json:"transaction_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.CreatePayment(ctx, &pb.CreatePaymentRequest{
		TransactionId: body.TransactionID,
		UserId:        userID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		return c.Status(grpcToHTTP(st.Code())).JSON(fiber.Map{
			"error": st.Message(),
		})
	}

	return c.Status(201).JSON(resp.Payment)
}


func (pc *PaymentController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.GetPayment(ctx, &pb.GetPaymentRequest{
		Id:     uint32(id),
		UserId: userID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		return c.Status(grpcToHTTP(st.Code())).JSON(fiber.Map{
			"error": st.Message(),
		})
	}

	return c.JSON(resp.Payment)
}


func (pc *PaymentController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.ListUserPayments(ctx, &pb.ListPaymentRequest{
		UserId: userID,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if resp.Payments == nil {
		resp.Payments = []*pb.Payment{}
	}

	return c.JSON(resp.Payments)
}

func (pc *PaymentController) ListAll(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.ListAllPayments(ctx, &emptypb.Empty{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if resp.Payments == nil {
		resp.Payments = []*pb.Payment{}
	}

	return c.JSON(resp.Payments)
}


func (pc *PaymentController) Pay(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := pc.Client.PayPayment(ctx, &pb.PayPaymentRequest{
		Id:     uint32(id),
		UserId: userID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		return c.Status(grpcToHTTP(st.Code())).JSON(fiber.Map{
			"error": st.Message(),
		})
	}

	return c.JSON(resp.Payment)
}


func grpcToHTTP(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return fiber.StatusNotFound
	case codes.PermissionDenied:
		return fiber.StatusForbidden
	case codes.InvalidArgument:
		return fiber.StatusBadRequest
	case codes.FailedPrecondition:
		return fiber.StatusConflict
	default:
		return fiber.StatusInternalServerError
	}
}
