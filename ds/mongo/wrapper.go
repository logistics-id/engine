package mongo

import (
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Config struct {
	Server     string
	Username   string
	Password   string
	Database   string
	Datasource string
	CtxTimeout time.Duration
}

var (
	defaultDB *mongo.Database
	logger    *zap.Logger
)

// setDefault fills in defaults if not explicitly provided.
func (c *Config) setDefault() {
	if c.Datasource == "" {
		c.Datasource = fmt.Sprintf("mongodb://%s:%s@%s", c.Username, c.Password, c.Server)
	}
	if c.CtxTimeout == 0 {
		c.CtxTimeout = 10 * time.Second
	}
}

// NewConnection sets up the MongoDB client and database.
func NewConnection(c *Config, l *zap.Logger) error {
	c.setDefault()

	logger = l

	clientOpts := options.Client().
		ApplyURI(c.Datasource).
		SetMonitor(monitoring())

	ctx := NewCtx(c.CtxTimeout)

	client, err := mongo.Connect(ctx, clientOpts)

	if err != nil {
		l.Fatal("MGO/CONN FAILED", zap.String("dsn", c.Datasource), zap.Any("config", c), zap.Error(err))
		return err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		l.Fatal("MGO/CONN FAILURE", zap.String("dsn", c.Datasource), zap.Any("config", c), zap.Error(err))
		return err
	}

	defaultDB = client.Database(c.Database)

	l.Info("MGO/CONN CONNECTED", zap.String("dsn", c.Datasource), zap.String("database", c.Database))

	return nil
}

func ConfigDefault(db string) *Config {
	return &Config{
		Server:   os.Getenv("MONGODB_SERVER"),
		Username: os.Getenv("MONGODB_AUTH_USERNAME"),
		Password: os.Getenv("MONGODB_AUTH_PASSWORD"),
		Database: db,
	}
}

func CloseConnection() error {
	return defaultDB.Client().Disconnect(nil)
}
