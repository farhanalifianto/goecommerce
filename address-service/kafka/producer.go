package kafka

import (
	"encoding/json"
	"log"
	"os"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
}

func NewProducer() *Producer {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "kafka:9092"
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll

	producer, err := sarama.NewSyncProducer([]string{broker}, config)
	if err != nil {
		log.Fatalf("❌ Failed to start Kafka producer: %v", err)
	}

	log.Println("✅ Kafka producer initialized for address-service")
	return &Producer{producer: producer}
}

func (p *Producer) PublishAddressCreatedEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("❌ Failed to marshal event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "address.created",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("❌ Failed to send Kafka message: %v", err)
		return
	}

	log.Printf("📤 Published address.created event: %v", string(data))
}
