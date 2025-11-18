package kafka

import (
	"encoding/json"
	"log"
	"user-service/model"

	"gorm.io/gorm"
)

type UserCreatedEvent struct {
	EventType string      `json:"event_type"`
	Data      UserPayload `json:"data"`
}

type UserPayload struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func HandleUserCreated(db *gorm.DB) func(data []byte) {
	return func(data []byte) {
		var event UserCreatedEvent

		if err := json.Unmarshal(data, &event); err != nil {
			log.Printf("❌ Failed decode event: %v", err)
			return
		}

		log.Printf("📥 Received user.created: %+v", event.Data)

		user := model.User{
			ID:    event.Data.ID,
			Email: event.Data.Email,
			Name:  event.Data.Name,
			Role:  event.Data.Role,
		}

		// Cek duplikat
		var existing model.User
		if err := db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
			log.Printf("User already exists (%s), skip", user.Email)
			return
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("Failed save user: %v", err)
			return
		}

		log.Printf("User saved: %s", user.Email)
	}
}
