package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/IBM/sarama"
)

func StartUserCreatedConsumer() {
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

			// TODO: Simpan user ke DB user-service
		case err := <-partitionConsumer.Errors():
			log.Printf("Kafka consumer error: %v", err)
		case <-context.Background().Done():
			return
		}
	}
}
