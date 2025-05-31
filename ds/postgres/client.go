package postgres

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

// Config holds the configuration parameters for connecting to a PostgreSQL database.
type Config struct {
	Server     string // Host or IP of the Postgres server
	Username   string // Database username
	Password   string // Database password
	Database   string // Database name
	Datasource string // Full DSN string (overrides Server/Username/Password/Database)
}

type Client struct {
	db     *bun.DB
	config *Config
	logger *zap.Logger
}

func NewClient(cfg *Config, l *zap.Logger) (*Client, error) {

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Datasource)))

	if err := sqldb.Ping(); err != nil {
		l.Fatal("PG/CONN FAILED", zap.String("dsn", cfg.Datasource), zap.Any("config", cfg), zap.Error(err))

		return nil, err
	}

	db := bun.NewDB(sqldb, pgdialect.New())

	// Add custom zap logger for query hooks
	db.AddQueryHook(&ZapQueryHook{Logger: l})

	l.Info("PG/CONN CONNECTED", zap.String("dsn", cfg.Datasource))

	return &Client{
		db:     db,
		config: cfg,
		logger: l,
	}, nil
}

func (c *Client) GetDB() *bun.DB {
	return c.db
}

func (c *Client) Close() error {
	client.logger.Info("PG/CONN CLOSED")

	return c.db.Close()
}
