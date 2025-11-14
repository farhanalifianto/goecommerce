package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"search-service/elasticsearch"
	"search-service/middleware"
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
			log.Printf(" Connected to Kafka at %s", broker)
			return consumerGroup
		}
		log.Printf("Waiting for Kafka... (%d/10) Error: %v", i, err)
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

		// Parse JSON event
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("❌ Failed to parse message: %v", err)
			continue
		}

		// Pastikan ada event_type
		eventType, ok := event["event_type"].(string)
        if !ok {
            log.Println("event_type missing or invalid")
            continue
        }

		// Pastikan ada data
		rawData, ok := event["data"]
		if !ok {
			log.Println("event.data missing")
			continue
		}

		data, ok := rawData.(map[string]interface{})
		if !ok {
			log.Println("event.data invalid format")
			continue
		}

		switch eventType {

		case "address_created", "address_updated":
			// Index address (Elastic hanya butuh data, bukan seluruh event)
			if err := h.esClient.IndexAddress(data); err != nil {
				log.Printf("❌ Failed to index: %v", err)
			}

		case "address_deleted":
			idValue, ok := data["id"]
			if !ok {
				log.Println("address_deleted missing id")
				continue
			}
			id := fmt.Sprintf("%v", idValue)

			if err := h.esClient.DeleteAddress(id); err != nil {
				log.Printf("Failed to delete index: %v", err)
			}

		default:
			log.Printf("Unknown event_type: %s", eventType)
		}

		session.MarkMessage(msg, "")
	}

	return nil
}

// ==== MAIN ====
func main() {
	broker := getEnv("KAFKA_BROKER", "kafka:9092")
	esHost := getEnv("ELASTICSEARCH_HOST", "http://elasticsearch:9200")

	log.Println("Starting search-service...")
	esClient := elasticsearch.NewElasticClient(esHost)


	// --- Jalankan server HTTP ---
	go func() {
		app := fiber.New()
		authWrapper := middleware.AuthMiddleware()
		// Panggil route terpisah
		routes.RegisterSearchRoutes(app,authWrapper, esClient)

		log.Println("HTTP server running on port 3004")
		if err := app.Listen(":3004"); err != nil {
			log.Fatalf("❌ HTTP server error: %v", err)
		}
	}()

	// --- Jalankan Kafka consumer ---
	consumerGroup := connectKafka(broker)
	handler := &ConsumerHandler{esClient: esClient}

	ctx := context.Background()
	topics := []string{
    "address.created",
    "address.updated",
    "address.deleted",
	}



	for {
		if err := consumerGroup.Consume(ctx, topics, handler); err != nil {
			log.Printf("❌ Kafka consume error: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}
