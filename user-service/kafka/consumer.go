package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"user-service/model"

	"github.com/IBM/sarama"
	"gorm.io/gorm"
)

// Terima DB agar bisa disimpan
func StartUserCreatedConsumer(db *gorm.DB) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "kafka:9092"
	}

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	client, err := sarama.NewConsumer([]string{broker}, config)
	if err != nil {
		log.Fatalf("Failed to start Kafka consumer: %v", err)
	}
	defer client.Close()

	partitionConsumer, err := client.ConsumePartition("user.created", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Failed to start partition consumer: %v", err)
	}
	defer partitionConsumer.Close()

	log.Println("👂 Listening for user.created events...")

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var user map[string]interface{}
			if err := json.Unmarshal(msg.Value, &user); err != nil {
				log.Printf("❌ Failed to parse user event: %v", err)
				continue
			}
			log.Printf("📥 Received user.created event: %+v", user)

			// Simpan ke DB
			u := model.User{
				ID:    uint(user["id"].(float64)),
				Email: user["email"].(string),
				Name:  user["name"].(string),
				Role:  user["role"].(string),
			}

			if err := db.Create(&u).Error; err != nil {
				log.Printf("❌ Failed to save user: %v", err)
			} else {
				log.Printf("✅ User saved to database: %v", u.Email)
			}

		case err := <-partitionConsumer.Errors():
			log.Printf("Kafka consumer error: %v", err)
		case <-context.Background().Done():
			return
		}
	}
}
