package kafka

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
)

var Producer sarama.SyncProducer

func InitProducer() {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "kafka:9092"
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	var err error
	for i := 1; i <= 5; i++ {
		Producer, err = sarama.NewSyncProducer([]string{broker}, config)
		if err == nil {
			log.Printf("Kafka producer connected to %s", broker)
			return
		}

		log.Printf("Failed to connect to Kafka (try %d/5): %v", i, err)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("❌ Could not connect to Kafka after 5 attempts: %v", err)
}

func PublishUserCreatedEvent(user interface{}) {
	if Producer == nil {
		log.Println(" Kafka producer is nil — event not sent")
		return
	}

	event := map[string]interface{}{
		"event_type": "user_created",
		"data":       user,
	}

	messageBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf(" Failed to marshal user event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "user.created",
		Value: sarama.ByteEncoder(messageBytes),
	}

	if _, _, err := Producer.SendMessage(msg); err != nil {
		log.Printf(" Failed to send Kafka message: %v", err)
	} else {
		log.Println("User created event sent to Kafka")
	}
}
