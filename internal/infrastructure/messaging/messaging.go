package messaging

import (
	"context"
	"fmt"

	"github.com/Amandasilvbr/products-crud/internal/domain/messaging"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Ensure RabbitMQClient implements the Publisher and Consumer interfaces at compile time
var _ messaging.Publisher = (*RabbitMQClient)(nil)

// RabbitMQClient wraps the RabbitMQ connection and channel
type RabbitMQClient struct {
	conn *amqp091.Connection
	ch   *amqp091.Channel
}

// NewRabbitMQClient creates and initializes a new RabbitMQ client
func NewRabbitMQClient(amqpURL string) (*RabbitMQClient, error) {
	zapLogger := zap.L()

	// Log the connection attempt (mask sensitive parts of the URL)
	zapLogger.Info("Trying to connect to RabbitMQ")

	// Establish a connection to the RabbitMQ server
	conn, err := amqp091.Dial(amqpURL)
	if err != nil {
		zapLogger.Error("Failed to connect to RabbitMQ", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Log a successful connection
	zapLogger.Info("Successfully connected to RabbitMQ")

	// Open a channel on the established connection
	ch, err := conn.Channel()
	if err != nil {
		zapLogger.Error("Failed to create RabbitMQ channel", zap.Error(err))
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Log successful channel creation
	zapLogger.Info("Successfully created RabbitMQ channel")

	return &RabbitMQClient{conn: conn, ch: ch}, nil
}

// DeclareQueue declares a queue on the RabbitMQ server
func (c *RabbitMQClient) DeclareQueue(queueName string) error {
	_, err := c.ch.QueueDeclare(
		queueName, // Name of the queue
		true,      // Durable
		false,     // Auto-delete
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	return err
}

// Publish sends a message to the specified queue
func (c *RabbitMQClient) Publish(ctx context.Context, queueName, body string) error {
	zapLogger := zap.L()
	err := c.ch.PublishWithContext(
		ctx,
		"",        // Exchange
		queueName, // Routing key (queue name)
		false,     // Mandatory
		false,     // Immediate
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		},
	)
	if err != nil {
		zapLogger.Error("Failed to publish message to RabbitMQ",
			zap.String("queue", queueName),
			zap.String("body", body),
			zap.Error(err))
		return err
	}
	zapLogger.Info("Successfully published message to RabbitMQ",
		zap.String("queue", queueName),
		zap.String("body", body))
	return nil
}

// Consume starts consuming messages from a specified queue
func (c *RabbitMQClient) Consume(queueName, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error) {
	return c.ch.Consume(
		queueName, // Name of the queue
		consumer,  // Consumer tag
		autoAck,   // Auto-acknowledgment
		exclusive, // Exclusive
		noLocal,   // No-local
		noWait,    // No-wait
		args,      // Arguments
	)
}

// CCloses the RabbitMQ channel and connection
func (c *RabbitMQClient) Close() {
	c.ch.Close()
	c.conn.Close()
}
