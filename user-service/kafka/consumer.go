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

func StartUserCreatedConsumer(db *gorm.DB) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "kafka:9092"
	}

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	var client sarama.Consumer
	var err error

	// 🔁 Coba connect ke Kafka sampai 5 kali
	for i := 1; i <= 5; i++ {
		client, err = sarama.NewConsumer([]string{broker}, config)
		if err == nil {
			log.Printf("✅ Connected to Kafka broker: %s", broker)
			break
		}
		log.Printf("⚠️ Failed to connect to Kafka (try %d/5): %v", i, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ Could not connect to Kafka after retries: %v", err)
	}
	defer client.Close()

	// 🧩 Pastikan topic "user.created" ada
	partitionConsumer, err := client.ConsumePartition("user.created", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("❌ Failed to start partition consumer: %v", err)
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

			u := model.User{
				ID:    uint(user["id"].(float64)),
				Email: user["email"].(string),
				Name:  user["name"].(string),
				Role:  user["role"].(string),
			}

			// 🧠 Cek dulu apakah user sudah ada
			var existing model.User
			if err := db.Where("id = ?", u.ID).First(&existing).Error; err == nil {
				log.Printf("⚠️ User already exists, skipping: %v", u.Email)
				continue
			}

			if err := db.Create(&u).Error; err != nil {
				log.Printf("❌ Failed to save user: %v", err)
			} else {
				log.Printf("✅ User saved to database: %v", u.Email)
			}

		case err := <-partitionConsumer.Errors():
			log.Printf("⚠️ Kafka consumer error: %v", err)
		case <-context.Background().Done():
			log.Println("🛑 Kafka consumer stopped.")
			return
		}
	}
}
