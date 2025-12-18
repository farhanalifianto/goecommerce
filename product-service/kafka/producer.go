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
			log.Println("Kafka producer initialized for product-service")
			return &Producer{producer: producer}
		}

		log.Printf("Waiting for Kafka... (%d/10) Error: %v", i, err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf("Failed to start Kafka producer after retries: %v", err)
	return nil
}

func (p *Producer) PublishProductCreatedEvent(event map[string]interface{}) {
	p.publish("product.created", event)
}

func (p *Producer) PublishProductUpdatedEvent(event map[string]interface{}) {
	p.publish("product.updated", event)
}

func (p *Producer) PublishProductDeletedEvent(event map[string]interface{}) {
	p.publish("product.deleted", event)
}
func (p *Producer) PublishCategoryCreatedEvent(event map[string]interface{}) {
	p.publish("category.created", event)
}

func (p *Producer) PublishCategoryUpdatedEvent(event map[string]interface{}) {
	p.publish("category.updated", event)
}

func (p *Producer) PublishCategoryDeletedEvent(event map[string]interface{}) {
	p.publish("category.deleted", event)
}
func (p *Producer) PublishStockUpdatedEvent(event map[string]interface{}) {
	p.publish("stock.updated", event)
}
func (p *Producer) publish(topic string, event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
		Timestamp: time.Now(),
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send Kafka message: %v", err)
		return
	}

	log.Printf(" Published %s: %s", topic, string(data))
}
