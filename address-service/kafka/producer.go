package kafka

import (
	"encoding/json"
	"log"
	"os"
	"time"

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

	var producer sarama.SyncProducer
	var err error

	for i := 1; i <= 10; i++ {
		producer, err = sarama.NewSyncProducer([]string{broker}, config)
		if err == nil {
			log.Println("Kafka producer initialized for address-service")
			return &Producer{producer: producer}
		}

		log.Printf("Waiting for Kafka... (%d/10) Error: %v", i, err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf("Failed to start Kafka producer after retries: %v", err)
	return nil
}

func (p *Producer) PublishAddressCreatedEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "address.created",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send Kafka message: %v", err)
		return
	}

	log.Printf("Published address.created event: %v", string(data))
}
func (p *Producer) PublishAddressUpdatedEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "address.updated",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send Kafka message: %v", err)
		return
	}

	log.Printf("Published address.updated event: %v", string(data))
}

func (p *Producer) PublishAddressDeletedEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "address.deleted",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send Kafka message: %v", err)
		return
	}

	log.Printf("Published address.deleted event: %v", string(data))
}
