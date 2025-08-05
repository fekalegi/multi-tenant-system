package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
)

type Publisher struct {
	rmq *Connection
	log zerolog.Logger
}

func NewPublisher(r *Connection, log zerolog.Logger) *Publisher {
	return &Publisher{rmq: r, log: log}
}

func (p *Publisher) Publish(tenantID string, payload map[string]interface{}) error {
	queue := fmt.Sprintf("tenant_%s_queue", tenantID)

	ch, err := p.rmq.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	data, _ := json.Marshal(payload)

	err = ch.Publish(
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			MessageId:   uuid.New().String(),
			Body:        data,
		},
	)
	return err
}

func (p *Publisher) PublishToTenantQueue(tenantID string, body []byte) error {
	channel, err := p.rmq.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer channel.Close()

	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)

	err = channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
