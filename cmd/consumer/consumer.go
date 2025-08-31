package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"time"

	"github.com/Amandasilvbr/products-crud/internal/config"
	"github.com/Amandasilvbr/products-crud/internal/infrastructure/messaging"

	"github.com/jordan-wright/email"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ProductEvent defines the structure for product-related events received from the message queue
type ProductEvent struct {
	Event            string `json:"event"`
	SKU              int    `json:"sku"`
	Name             string `json:"name"`
	ResponsibleEmail string `json:"responsible_email"`
}

// batchItem holds both the deserialized event and the original message,
// which is necessary for acknowledging or rejecting the message after processing
type batchItem struct {
	event ProductEvent
	msg   amqp091.Delivery
}

// Consumer represents a RabbitMQ consumer that processes product events
type Consumer struct {
	logger    *zap.Logger
	rabbitMQ  *messaging.RabbitMQClient
	queueName string
}

// NewConsumer creates and initializes a new RabbitMQ consumer
func NewConsumer(logger *zap.Logger, rabbitMQURL, queueName string) (*Consumer, error) {
	if rabbitMQURL == "" {
		logger.Error("RabbitMQ URL is missing")
		return nil, fmt.Errorf("RABBITMQ_URL is missing")
	}
	if queueName == "" {
		logger.Error("Queue name is missing")
		return nil, fmt.Errorf("queue name is missing")
	}

	logger.Info("Initializing RabbitMQ consumer")

	// Initialize RabbitMQ client
	rabbitMQ, err := messaging.NewRabbitMQClient(rabbitMQURL)
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	logger.Info("Connected to RabbitMQ successfully")

	// Declare the queue
	err = rabbitMQ.DeclareQueue(queueName)
	if err != nil {
		logger.Error("Failed to declare queue", zap.String("queue", queueName), zap.Error(err))
		rabbitMQ.Close()
		return nil, fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}
	logger.Info("Queue declared successfully", zap.String("queue", queueName))

	return &Consumer{
		logger:    logger,
		rabbitMQ:  rabbitMQ,
		queueName: queueName,
	}, nil
}

// Start begins the message consumption loop
// It processes messages in batches, sending an email summary when a batch is full or a timeout occurs
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting consumer for queue", zap.String("queue", c.queueName))

	msgs, err := c.rabbitMQ.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		c.logger.Error("Failed to consume messages", zap.String("queue", c.queueName), zap.Error(err))
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	c.logger.Info("Consumer started, waiting for messages...")

	// Configure batch processing parameters
	const batchSize = 100                // Maximum number of messages per batch
	const batchTimeout = 5 * time.Second // Maximum time to wait before processing a batch
	var batch []batchItem                // Slice to accumulate messages
	batchTimer := time.NewTimer(batchTimeout)
	defer batchTimer.Stop()

	// Main processing loop that listens for messages, context cancellation, or timer events
	for {
		select {
		// Handle shutdown signal
		case <-ctx.Done():
			c.logger.Info("Consumer interrupted due to context cancellation")
			// Process any pending messages in the batch before shutting down
			if len(batch) > 0 {
				events := make([]ProductEvent, len(batch))
				for i, item := range batch {
					events[i] = item.event
				}
				if err := c.sendEmail(events); err != nil {
					c.logger.Error("Failed to send email for pending batch", zap.Error(err))
					// Requeue all messages in the batch if email sending fails
					for _, item := range batch {
						item.msg.Nack(false, true)
					}
				} else {
					// Acknowledge all messages in the batch upon success
					for _, item := range batch {
						item.msg.Ack(false)
					}
				}
			}
			return ctx.Err()

		// Handle incoming messages from the queue
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Error("Message channel closed")
				// If the channel closes, process any pending batch
				if len(batch) > 0 {
					events := make([]ProductEvent, len(batch))
					for i, item := range batch {
						events[i] = item.event
					}
					if err := c.sendEmail(events); err != nil {
						c.logger.Error("Failed to send email for pending batch", zap.Error(err))
						// Requeue all messages
						for _, item := range batch {
							item.msg.Nack(false, true)
						}
					} else {
						// Acknowledge all messages
						for _, item := range batch {
							item.msg.Ack(false)
						}
					}
				}
				return fmt.Errorf("message channel closed")
			}

			c.logger.Info("Message received from RabbitMQ", zap.String("body", string(msg.Body)))

			// Deserialize the JSON message body into a ProductEvent struct
			var event ProductEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				c.logger.Error("Failed to deserialize message", zap.String("body", string(msg.Body)), zap.Error(err))
				msg.Nack(false, true) // Requeue the message if deserialization fails
				continue
			}

			c.logger.Info("Processing event",
				zap.String("event", event.Event),
				zap.Int("sku", event.SKU),
				zap.String("responsible_email", event.ResponsibleEmail))

			// Add the successfully processed event to the current batch
			batch = append(batch, batchItem{event: event, msg: msg})

			// If the batch reaches its maximum size, process it immediately
			if len(batch) >= batchSize {
				events := make([]ProductEvent, len(batch))
				for i, item := range batch {
					events[i] = item.event
				}
				if events[0].ResponsibleEmail != "" {
					if err := c.sendEmail(events); err != nil {
						c.logger.Error("Failed to send email for batch", zap.Error(err))
						// Requeue all messages on failure
						for _, item := range batch {
							item.msg.Nack(false, true)
						}
						batch = nil
						batchTimer.Reset(batchTimeout)
						continue
					}
					c.logger.Info("Batch email sent successfully", zap.Int("batch_size", len(batch)))
				} else {
					c.logger.Warn("No ResponsibleEmail provided for batch", zap.Int("batch_size", len(batch)))
				}
				// Acknowledge all messages in the batch
				for _, item := range batch {
					item.msg.Ack(false)
				}
				batch = nil
				batchTimer.Reset(batchTimeout)
			}

		// Handle the batch timeout
		case <-batchTimer.C:
			// If there are any messages in the batch when the timer fires, process them
			if len(batch) > 0 {
				events := make([]ProductEvent, len(batch))
				for i, item := range batch {
					events[i] = item.event
				}
				if events[0].ResponsibleEmail != "" {
					if err := c.sendEmail(events); err != nil {
						c.logger.Error("Failed to send email for batch", zap.Error(err))
						// Requeue all messages on failure
						for _, item := range batch {
							item.msg.Nack(false, true)
						}
						batch = nil
						batchTimer.Reset(batchTimeout)
						continue
					}
					c.logger.Info("Batch email sent successfully after timeout", zap.Int("batch_size", len(batch)))
				} else {
					c.logger.Warn("No ResponsibleEmail provided for batch", zap.Int("batch_size", len(batch)))
				}
				// Acknowledge all messages in the batch
				for _, item := range batch {
					item.msg.Ack(false)
				}
				batch = nil
			}
			// Restart the timer
			batchTimer.Reset(batchTimeout)
		}
	}
}

// sendEmail constructs and sends an email summarizing a batch of product events
func (c *Consumer) sendEmail(events []ProductEvent) error {
	if len(events) == 0 {
		return nil // Avoid sending an empty email
	}

	// Load application configuration to get SMTP settings
	cfg, err := config.New()
	if err != nil {
		c.logger.Error("Failed to load configuration", zap.Error(err))
		return err
	}

	// Validate that required SMTP configuration is present
	if cfg.SMTPHost == "" || cfg.SMTPPort == "" || cfg.SMTPFrom == "" {
		err := fmt.Errorf("missing SMTP configuration: host=%s, port=%s, from=%s", cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)
		c.logger.Error("Invalid SMTP configuration", zap.Error(err))
		return err
	}

	e := email.NewEmail()
	e.From = cfg.SMTPFrom
	e.To = []string{events[0].ResponsibleEmail} // Assumes all events in a batch have the same responsible email

	// Map event types to user-friendly descriptions for singular and plural forms
	eventMessages := map[string]map[string]string{
		"product_created": {"singular": "criado", "plural": "criados"},
		"product_updated": {"singular": "atualizado", "plural": "atualizados"},
		"product_deleted": {"singular": "deletado", "plural": "deletados"},
	}

	// Count occurrences of each event type to build a summary
	eventCounts := make(map[string]int)
	for _, event := range events {
		eventCounts[event.Event]++
	}

	// Dynamically create the email subject based on the event counts
	var subjectSummary string
	for eventType, count := range eventCounts {
		actions, ok := eventMessages[eventType]
		if !ok {
			// Fallback for unknown event types
			if count == 1 {
				subjectSummary += fmt.Sprintf("%d Produto foi %s, ", count, eventType)
			} else {
				subjectSummary += fmt.Sprintf("%d Produtos foram %s, ", count, eventType)
			}
			continue
		}
		if count == 1 {
			subjectSummary += fmt.Sprintf("%d Produto foi %s, ", count, actions["singular"])
		} else {
			subjectSummary += fmt.Sprintf("%d Produtos foram %s, ", count, actions["plural"])
		}
	}
	subjectSummary = subjectSummary[:len(subjectSummary)-2] // Remove trailing comma and space
	e.Subject = fmt.Sprintf("Resumo de Notificações de Produtos: %s", subjectSummary)

	// Build the plain text version of the email body
	var textBody string
	if len(events) == 1 {
		textBody = "Olá,\n\nSegue o resumo da alteração no sistema:\n\n"
	} else {
		textBody = "Olá,\n\nSegue o resumo das alterações no sistema:\n\n"
	}
	for _, event := range events {
		actions, ok := eventMessages[event.Event]
		action := event.Event // Fallback
		if ok {
			action = actions["singular"] // Use singular form for individual list items
		}
		textBody += fmt.Sprintf("- Product %s (SKU: %d) was %s\n", event.Name, event.SKU, action)
	}
	textBody += fmt.Sprintf("\nData: %s\n\nAtenciosamente,\nEquipe de Produtos", time.Now().Format("02/01/2006 15:04:05"))
	e.Text = []byte(textBody)

	// Build the HTML version of the email body with inline styles
	var htmlBody string
	if len(events) == 1 {
		htmlBody = `
    <html>
    <head>
        <style>
            body { font-family: Arial, sans-serif; color: #333; }
            .container { max-width: 600px; margin: 0 auto; padding: 20px; }
            .header { background-color: #f5f5f5; padding: 10px; text-align: left; }
            .header h2 { text-align: center; margin-top: 10px; }
            .content { padding: 20px; }
            .footer { background-color: #f5f5f5; font-size: 12px; text-align: center; margin-top: 20px; padding: 15px; }
            .image-footer { max-width: 150px; margin: 10px auto 20px; display: block; padding: 0 15px; }
        </style>
    </head>
    <body>
        <div class="container">
            <div class="header">
                <h2>Resumo de Notificações de Produtos</h2>
            </div>
            <div class="content">
                <p>Olá,</p>
                <p>Segue o resumo da alteração no sistema:</p>
                <ul>
    `
	} else {
		htmlBody = `
    <html>
    <head>
        <style>
            body { font-family: Arial, sans-serif; color: #333; }
            .container { max-width: 600px; margin: 0 auto; padding: 20px; }
            .header { background-color: #f5f5f5; padding: 10px; text-align: left; }
            .header h2 { text-align: center; margin-top: 10px; }
            .content { padding: 20px; }
            .footer { background-color: #f5f5f5; font-size: 12px; text-align: center; margin-top: 20px; padding: 15px; }
            .image-footer { max-width: 150px; margin: 10px auto 20px; display: block; padding: 0 15px; }
        </style>
    </head>
    <body>
        <div class="container">
            <div class="header">
                <h2>Resumo de Notificações de Produtos</h2>
            </div>
            <div class="content">
                <p>Olá,</p>
                <p>Segue o resumo das alterações no sistema:</p>
                <ul>
    `
	}
	for _, event := range events {
		actions, ok := eventMessages[event.Event]
		action := event.Event
		if ok {
			action = actions["singular"]
		}
		htmlBody += fmt.Sprintf(
			"<li>Produto <strong>%s</strong> (SKU: %d) foi %s</li>",
			event.Name, event.SKU, action,
		)
	}
	htmlBody += fmt.Sprintf(`
                </ul>
                <p><strong>Data:</strong> %s</p>
                <p>Se precisar de mais informações, entre em contato com nossa equipe.</p>
            </div>
            <div class="footer">
                <p>Atenciosamente,<br>Equipe de Produtos</p>
                <img src="https://i.ibb.co/M5gN8pXr/karhub-logo-01-main-e1721220516367.png" alt="Logo da Empresa" class="image-footer">
            </div>
        </div>
    </body>
    </html>
    `, time.Now().Format("02/01/2006 15:04:05"))
	e.HTML = []byte(htmlBody)

	c.logger.Info("Trying to send batch email",
		zap.String("to", events[0].ResponsibleEmail),
		zap.String("subject", e.Subject),
		zap.Int("event_count", len(events)))

	// Send the email using the configured SMTP server
	smtpAddr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	err = e.Send(
		smtpAddr,
		smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost),
	)
	if err != nil {
		c.logger.Error("Failed to send batch email",
			zap.String("to", events[0].ResponsibleEmail),
			zap.Error(err))
		return err
	}

	c.logger.Info("Batch email sent successfully",
		zap.String("to", events[0].ResponsibleEmail),
		zap.Int("event_count", len(events)))
	return nil
}

// closes the connection to RabbitMQ
func (c *Consumer) Close() {
	if c.rabbitMQ != nil {
		c.rabbitMQ.Close()
		c.logger.Info("Connection to RabbitMQ closed")
	}
}
