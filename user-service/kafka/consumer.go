package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"
	"user-service/model"

	"github.com/IBM/sarama"
	"gorm.io/gorm"
)

// =============== EVENT STRUCT ===============

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

// =============== CONSUMER STARTER ===============

func StartUserCreatedConsumer(db *gorm.DB) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "kafka:9092"
	}

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	var client sarama.Consumer
	var err error

	// Retry Kafka connection
	for i := 1; i <= 5; i++ {
		client, err = sarama.NewConsumer([]string{broker}, config)
		if err == nil {
			log.Printf("Connected to Kafka broker: %s", broker)
			break
		}
		log.Printf("Failed to connect to Kafka (try %d/5): %v", i, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("Could not connect to Kafka after retries: %v", err)
	}
	defer client.Close()

	// Open partition consumer
	partitionConsumer, err := client.ConsumePartition("user.created", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Failed to start partition consumer: %v", err)
	}
	defer partitionConsumer.Close()

	log.Println("📡 Listening for user.created events...")

	// =============== CONSUMER LOOP ===============
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			handleUserCreatedEvent(msg.Value, db)

		case err := <-partitionConsumer.Errors():
			log.Printf("Kafka consumer error: %v", err)

		case <-context.Background().Done():
			log.Println("Kafka consumer stopped.")
			return
		}
	}
}

// =============== HANDLER ===============

func handleUserCreatedEvent(raw []byte, db *gorm.DB) {
	var event UserCreatedEvent

	// Parse JSON
	if err := json.Unmarshal(raw, &event); err != nil {
		log.Printf("❌ Failed to parse event JSON: %v", err)
		return
	}

	log.Printf("📥 Received user.created event: %+v", event)

	// Extract user
	data := event.Data

	user := model.User{
		ID:    data.ID,
		Email: data.Email,
		Name:  data.Name,
		Role:  data.Role,
	}

	// Check duplicate email
	var existing model.User
	if err := db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
		log.Printf("⚠️ User already exists (%s), skipping.", user.Email)
		return
	}

	// Save to DB
	if err := db.Create(&user).Error; err != nil {
		log.Printf("❌ Failed to save user: %v", err)
	} else {
		log.Printf("✅ User saved to DB: %v", user.Email)
	}
}
