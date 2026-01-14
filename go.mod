module github.com/logistics-id/engine

go 1.24.3

require (
	github.com/gomodule/redigo v1.9.2
	github.com/gorilla/websocket v1.5.3
	github.com/logistics-id/engine/broker/rabbitmq v0.0.19-dev
	github.com/logistics-id/engine/common v0.0.19-dev
	github.com/rabbitmq/amqp091-go v1.10.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
)
