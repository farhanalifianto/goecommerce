package kafka

import (
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
)

type Consumer struct {
    consumer sarama.Consumer
}

func NewConsumer() *Consumer {
    broker := os.Getenv("KAFKA_BROKER")
    if broker == "" {
        broker = "kafka:9092"
    }

    config := sarama.NewConfig()
    config.Consumer.Return.Errors = true

    var client sarama.Consumer
    var err error

    for i := 1; i <= 10; i++ {
        client, err = sarama.NewConsumer([]string{broker}, config)
        if err == nil {
            log.Println("Kafka consumer initialized")
            return &Consumer{consumer: client}
        }

        log.Printf("Waiting for Kafka consumer... (%d/10) Error: %v", i, err)
        time.Sleep(5 * time.Second)
    }

    log.Fatalf("Failed to start Kafka consumer: %v", err)
    return nil
}

// Generic consume function
func (c *Consumer) Consume(topic string, handler func([]byte)) {
    pc, err := c.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
    if err != nil {
        log.Fatalf("❌ Failed to consume topic %s: %v", topic, err)
    }

    log.Printf("📡 Listening on topic %s ...", topic)

    go func() {
        for {
            select {
            case msg := <-pc.Messages():
                handler(msg.Value)

            case err := <-pc.Errors():
                log.Printf("Kafka consumer error: %v", err)
            }
        }
    }()
}
