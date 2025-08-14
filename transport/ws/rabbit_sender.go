// Package ws provides WebSocket transport logic for message sending via RabbitMQ.
package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/logistics-id/engine/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RMQSender struct {
	PodID    string
	Broker   *rabbitmq.Client
	Hub      *Hub
	Registry Registry
	Logger   *zap.Logger
}

func (s *RMQSender) getKey(pod string) string {
	return fmt.Sprintf("ws.send.%s", pod)
}

func (s *RMQSender) SendToUser(ctx context.Context, userID string, msg []byte) error {
	pods, err := s.Registry.GetUserPods(ctx, userID)

	logger := s.Logger.With(zap.String("user_id", userID))

	if err != nil {
		logger.Error("failed to get user pods", zap.Error(err))
		return err
	} else {
		for _, pod := range pods {
			log := logger.With(zap.String("pod", pod))

			if pod == s.PodID {
				log.Debug("sent to local user")
				s.Hub.SendLocal(userID, msg)
			} else {
				log.Debug("publishing to routing key", zap.String("routingKey", s.getKey(pod)))

				err = s.Broker.Publish(ctx, s.getKey(pod), msg)
				if err != nil {
					log.Error("failed to publish to remote pod", zap.Error(err))
					return err
				}

				log.Debug("published to remote pod")
			}
		}
	}

	return nil
}

func NewRMQSender(podID string, broker *rabbitmq.Client, hub *Hub, registry Registry, logger *zap.Logger) *RMQSender {
	key := fmt.Sprintf("ws.send.%s", podID)

	logger = logger.With(zap.String("pod_id", podID))

	err := broker.Subscribe(
		key,
		key,
		func(data []byte, msg amqp.Delivery) error {
			var env Envelope
			if err := json.Unmarshal(data, &env); err != nil {
				logger.Error("failed to unmarshal message", zap.Error(err))
				return err
			}

			err := hub.SendLocal(env.UserID, data)
			if err != nil {
				logger.Error("Failed send to local", zap.Error(err))
				return err
			}

			return msg.Ack(true)
		},
	)
	if err != nil {
		logger.Error("Failed to subscribe to RMQ topic", zap.String("topic", key), zap.Error(err))
		return nil
	}

	return &RMQSender{
		PodID:    podID,
		Broker:   broker,
		Hub:      hub,
		Registry: registry,
		Logger:   logger.With(zap.String("pod_id", podID)),
	}
}
