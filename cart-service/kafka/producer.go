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
			log.Println("Kafka producer initialized for cart-service")
			return &Producer{producer: producer}
		}

		log.Printf("Waiting for Kafka... (%d/10) Error: %v", i, err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf(" Failed to start Kafka producer after retries: %v", err)
	return nil
}


func (p *Producer) PublishCartPaidEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal cart.paid event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "cart.paid",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send cart.paid Kafka message: %v", err)
		return
	}

	log.Printf("📤 Published cart.paid event: %v", string(data))
}

func (p *Producer) PublishCartItemAddedEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal cart.item.added: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "cart.item.added",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send cart.item.added Kafka message: %v", err)
		return
	}

	log.Printf("Published cart.item.added event: %v", string(data))
}

func (p *Producer) PublishCartItemRemovedEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf(" Failed to marshal cart.item.removed: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: "cart.item.removed",
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("❌ Failed to send cart.item.removed Kafka message: %v", err)
		return
	}

	log.Printf("Published cart.item.removed event: %v", string(data))
}
