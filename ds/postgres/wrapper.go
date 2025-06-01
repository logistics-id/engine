package postgres

import (
	"errors"
	"fmt"
	"os"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// internal shared client singleton
var (
	client                  *Client
	ErrClientNotInitialized = errors.New("db client not initialized; call NewConnection first")
)

// NewConnection initializes a new Bun DB connection and sets it as the default.
// It also adds a zap logger hook and optionally a verbose logger if debug is true.
func NewConnection(c *Config, l *zap.Logger) (err error) {
	l = l.With(
		zap.String("component", "ds.postgres"),
		zap.String("database", c.Database),
	)

	client, err = NewClient(c, l)

	return
}

// ConfigDefault creating an config readed from .env file
// make sure you load the env file in your init app
func ConfigDefault(db string) *Config {
	c := &Config{
		Server:   os.Getenv("POSTGRES_SERVER"),
		Username: os.Getenv("POSTGRES_AUTH_USERNAME"),
		Password: os.Getenv("POSTGRES_AUTH_PASSWORD"),
		Database: db,
	}

	c.Datasource = fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		c.Username, c.Password, c.Server, c.Database,
	)

	return c
}

// GetDB returns the globally initialized *bun.DB instance.
// You should call NewConnection first before calling GetDB.
func GetDB() *bun.DB {
	return client.GetDB()
}

// CloseConnection closes the default client connection.
func CloseConnection() error {
	if client.db == nil {
		return ErrClientNotInitialized
	}

	return client.Close()
}
