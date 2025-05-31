package engine_test

import (
	"os"
	"testing"

	"github.com/logistics-id/engine"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {

	c := &engine.Config{
		Name:    "service.test",
		IsDev:   true,
		Version: os.Getenv("APP_VERSION"),
	}

	engine.Start(c)

	engine.Logger.Info("loggin is running on dev mode")

	assert.Equal(t, c.Name, engine.Service.Name)
	assert.Equal(t, c.IsDev, c.IsDev)
}
