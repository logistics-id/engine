package ws

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type MQSender struct {
	PodID    string
	Hub      *Hub
	Channel  *amqp.Channel
	Exchange string
	Registry Registry
	Logger   *zap.Logger
}

func (s *MQSender) SendToUser(ctx context.Context, userID string, msg []byte) error {
	pods, err := s.Registry.GetUserPods(ctx, userID)
	if err != nil {
		s.Logger.Error("failed to get user pods", zap.String("userID", userID), zap.Error(err))
		return err
	}
	for _, pod := range pods {
		if pod == s.PodID {
			s.Hub.SendLocal(userID, msg)
			s.Logger.Debug("sent to local user", zap.String("userID", userID))
		} else {
			err = s.Channel.PublishWithContext(ctx, s.Exchange, "ws.send."+pod, false, false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        msg,
				})
			if err != nil {
				s.Logger.Error("failed to publish to remote pod", zap.String("userID", userID), zap.String("pod", pod), zap.Error(err))
				return err
			}
			s.Logger.Debug("published to remote pod", zap.String("userID", userID), zap.String("pod", pod))
		}
	}
	return nil
}

// StartConsumer sets up a RabbitMQ consumer that delivers incoming messages to local connections.
func StartConsumer(ch *amqp.Channel, queueName string, hub *Hub) error {
	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			var env Envelope
			if err := json.Unmarshal(d.Body, &env); err != nil {
				continue
			}
			_ = hub.SendLocal(env.UserID, d.Body)
		}
	}()
	return nil
}

// NewRabbitSender creates an MQSender and starts the consumer for this pod.
func NewRabbitSender(ch *amqp.Channel, podID string, hub *Hub, registry Registry, logger *zap.Logger) (*MQSender, error) {
	exchange := "ws"
	routingKey := "ws.send." + podID
	queue := routingKey

	// Declare the exchange and queue
	if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("exchange declare failed: %w", err)
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("queue declare failed: %w", err)
	}
	if err := ch.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
		return nil, fmt.Errorf("queue bind failed: %w", err)
	}

	// Start RabbitMQ consumer for incoming cross-pod messages
	if err := StartConsumer(ch, queue, hub); err != nil {
		return nil, fmt.Errorf("start consumer failed: %w", err)
	}

	// Create and return the MQSender
	return &MQSender{
		PodID:    podID,
		Hub:      hub,
		Channel:  ch,
		Exchange: exchange,
		Registry: registry,
		Logger:   logger,
	}, nil
}
