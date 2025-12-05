package controller

import (
	"address-service/grpc_client"
	pb "address-service/proto/address"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AddressController struct {
	Client pb.AddressServiceClient
	UserClient   *grpc_client.UserClient
}

func (ac *AddressController) List(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userInfo, err := ac.UserClient.GetUserEmail(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	resp, err := ac.Client.ListAddresses(ctx, &pb.ListAddressRequest{
		OwnerId: uint32(userID),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
    type AddressResponse struct {
        ID      uint32 `json:"id"`
        Name    string `json:"name"`
        Desc    string `json:"desc"`
        OwnerID string `json:"owner_id"`
    }

    var out []AddressResponse
    for _, addr := range resp.Addresses {
        out = append(out, AddressResponse{
            ID:      addr.Id,
            Name:    addr.Name,
            Desc:    addr.Desc,
            OwnerID: userInfo.Email, // ganti owner_id angka → email
        })
    }

    return c.JSON(out)
}

func (ac *AddressController) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	userID := c.Locals("user_id").(uint32) // dari JWT

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.GetAddress(ctx, &pb.GetAddressRequest{
		Id:       uint32(id),
		OwnerId:  userID, // kirim ke gRPC untuk validasi
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.PermissionDenied {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "you are not the owner of this address",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	userInfo, err := ac.UserClient.GetUserEmail(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	out := fiber.Map{
		"id":         resp.Address.Id,
		"name":       resp.Address.Name,
		"desc":       resp.Address.Desc,
		"owner_id":   userInfo.Email,
	}

	return c.JSON(out)
}

func (ac *AddressController) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint32)

	var body struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.CreateAddress(ctx, &pb.CreateAddressRequest{
		Name:     body.Name,
		Desc:     body.Desc,
		OwnerId:  uint32(userID),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(resp.Address)
}

func (ac *AddressController) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.UpdateAddress(ctx, &pb.UpdateAddressRequest{
		Id:   uint32(id),
		Name: body.Name,
		Desc: body.Desc,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp.Address)
}

func (ac *AddressController) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := ac.Client.DeleteAddress(ctx, &pb.DeleteAddressRequest{Id: uint32(id)})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}

func (ac *AddressController) GetAllAddresses(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Ambil semua address dari AddressService
    resp, err := ac.Client.GetAllAddresses(ctx, &emptypb.Empty{})
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    // Struct untuk response JSON
    type AddressResponse struct {
        ID         uint32 `json:"id"`
        Name       string `json:"name"`
        Desc       string `json:"desc"`
        OwnerEmail string `json:"owner_email"`
    }

    var out []AddressResponse

    // Loop semua address
    for _, addr := range resp.Addresses {
        // --- Panggil gRPC ke UserService untuk ambil email ---
        userResp, err := ac.UserClient.GetUserEmail(addr.OwnerId)
        if err != nil {
            // Kalau error, bisa lanjut tapi kasih tanda error di email-nya
            fmt.Printf("failed to get email for owner_id=%d: %v\n", addr.OwnerId, err)
            out = append(out, AddressResponse{
                ID:         addr.Id,
                Name:       addr.Name,
                Desc:       addr.Desc,
                OwnerEmail: fmt.Sprintf("error: %v", err),
            })
            continue
        }

        // Tambahkan hasil ke response
        out = append(out, AddressResponse{
            ID:         addr.Id,
            Name:       addr.Name,
            Desc:       addr.Desc,
            OwnerEmail: userResp.Email,
        })
    }

    return c.JSON(out)
}
func NewAddressController() *AddressController {
	addrConn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
	if err != nil {
		panic("failed to connect to address gRPC: " + err.Error())
	}

	addrClient := pb.NewAddressServiceClient(addrConn)
	userClient := grpc_client.NewUserClient()

	return &AddressController{
		Client: addrClient,
		UserClient:    userClient,
	}
}