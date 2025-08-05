package rabbitmq

import (
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
)

type Connection struct {
	conn *amqp.Connection
	log  zerolog.Logger
}

func NewConnection(url string, log zerolog.Logger) *Connection {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}

	log.Info().Msg("Connected to RabbitMQ")
	return &Connection{
		conn: conn,
		log:  log,
	}
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	return c.conn.Channel()
}

func (c *Connection) Close() {
	if err := c.conn.Close(); err != nil {
		c.log.Error().Err(err).Msg("Failed to close RabbitMQ connection")
	}
}
