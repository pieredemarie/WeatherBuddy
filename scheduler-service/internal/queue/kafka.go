package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/segmentio/kafka-go"
)

const NotificationJobsTopic = "notification-jobs"

type NotificationJob struct {
	UserID       int     `json:"user_id"`
	ContactType  int     `json:"contact_type"`
	ContactValue int     `json:"contact_value"`
	City         string  `json:"city"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},

			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, job NotificationJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal notification job: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(strconv.Itoa(job.UserID)),
		Value: payload,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
