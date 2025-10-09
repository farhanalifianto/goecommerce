package controller

import (
	"time"

	"user-service/model"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct {
	DB        *gorm.DB
	JWTSecret string
}

func (uc *UserController) Register(c *fiber.Ctx) error {
	in := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role	 string `json:"role"`

	}{}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}
	if in.Email == "" || in.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "email and password required"})
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	user := model.User{Email: in.Email, Password: string(hashed), Name: in.Name, Role: "user"}
	if err := uc.DB.Create(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "email already used"})
	}
	return c.JSON(fiber.Map{"id": user.ID, "email": user.Email, "name": user.Name})
}

func (uc *UserController) Login(c *fiber.Ctx) error {
	in := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}
	var user model.User
	if err := uc.DB.Where("email = ?", in.Email).First(&user).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(uc.JWTSecret))

	return c.JSON(fiber.Map{"access_token": signed})
}

func (uc *UserController) Me(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var user model.User
	uc.DB.First(&user, userID)
	return c.JSON(fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

