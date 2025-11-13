package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"search-service/elasticsearch"
	"search-service/routes"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
)

// ==== Helper Env ====
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ==== Connect Kafka with retry ====
func connectKafka(broker string) sarama.ConsumerGroup {
	var consumerGroup sarama.ConsumerGroup
	var err error

	config := sarama.NewConfig()
	config.Version = sarama.V3_4_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	for i := 1; i <= 10; i++ {
		consumerGroup, err = sarama.NewConsumerGroup([]string{broker}, "search-service-group", config)
		if err == nil {
			log.Printf("✅ Connected to Kafka at %s", broker)
			return consumerGroup
		}
		log.Printf("⏳ Waiting for Kafka... (%d/10) Error: %v", i, err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf("❌ Failed to connect Kafka after retries: %v", err)
	return nil
}

// ==== Consumer Handler ====
type ConsumerHandler struct {
	esClient *elasticsearch.ElasticClient
}

func (h *ConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Printf("📨 Received message: %s", string(msg.Value))

		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("❌ Failed to parse message: %v", err)
			continue
		}

		eventType, _ := event["event_type"].(string)
		data, _ := event["data"].(map[string]interface{})
		log.Printf("🔍 Event Type: %s | Data: %v", eventType, data)

		if eventType == "address_created" {
			if err := h.esClient.IndexAddress(data); err != nil {
				log.Printf("❌ Failed to index to Elasticsearch: %v", err)
			}
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

// ==== MAIN ====
func main() {
	broker := getEnv("KAFKA_BROKER", "kafka:9092")
	esHost := getEnv("ELASTICSEARCH_HOST", "http://elasticsearch:9200")

	log.Println("🚀 Starting search-service...")
	esClient := elasticsearch.NewElasticClient(esHost)

	// --- Jalankan server HTTP ---
	go func() {
		app := fiber.New()

		// Panggil route terpisah
		routes.RegisterSearchRoutes(app, esClient)

		log.Println("🌐 HTTP server running on port 3004")
		if err := app.Listen(":3004"); err != nil {
			log.Fatalf("❌ HTTP server error: %v", err)
		}
	}()

	// --- Jalankan Kafka consumer ---
	consumerGroup := connectKafka(broker)
	handler := &ConsumerHandler{esClient: esClient}

	ctx := context.Background()
	topics := []string{"address_events"}

	for {
		if err := consumerGroup.Consume(ctx, topics, handler); err != nil {
			log.Printf("❌ Kafka consume error: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}
