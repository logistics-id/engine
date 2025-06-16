package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/logistics-id/engine/broker/rabbitmq"
)

// Sender is responsible for sending messages to users via RabbitMQ and tracking user presence.
type Sender struct {
	Rabbit   *rabbitmq.Client  // RabbitMQ client for publishing messages
	Registry *PresenceRegistry // Registry to track which pods users are connected to
}

// Envelope wraps the user ID and payload for message delivery.
type Envelope struct {
	UserID  string `json:"user_id"` // Target user ID
	Payload any    `json:"payload"` // Message payload
}

// SendToUser sends a payload to all pods where the user is currently connected.
// It retrieves the user's active pod IDs from the registry and publishes the message to each pod's routing key.
func (s *Sender) SendToUser(ctx context.Context, userID string, payload any) error {
	podIDs, err := s.Registry.GetPods(userID)
	if err != nil || len(podIDs) == 0 {
		return fmt.Errorf("no active pods for user %s", userID)
	}

	body, err := json.Marshal(Envelope{
		UserID:  userID,
		Payload: payload,
	})
	if err != nil {
		return err
	}

	for _, podID := range podIDs {
		routingKey := fmt.Sprintf("ws.send.%s", podID)
		s.Rabbit.Publish(ctx, routingKey, body)
	}
	return nil
}
