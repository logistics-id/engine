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
	l = l.With(
		zap.String("dsn", cfg.Datasource),
		zap.String("database", cfg.Database),
	)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Datasource)))

	if err := sqldb.Ping(); err != nil {
		l.Error("PG/CONN FAILED", zap.Error(err))

		return nil, err
	}

	db := bun.NewDB(sqldb, pgdialect.New())

	// Add custom zap logger for query hooks
	db.AddQueryHook(&ZapQueryHook{Logger: l})

	l.Info("PG/CONN CONNECTED", zap.String("action", "connection"))

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
