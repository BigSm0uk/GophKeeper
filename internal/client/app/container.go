package app

import (
	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/config"
	"go.uber.org/zap"
)

// Container хранит зависимости клиента.
type Container struct {
	Logger *zap.Logger
	Config *config.ClientConfig
	API    *api.Client
}

func NewContainer(logger *zap.Logger, cfg *config.ClientConfig, client *api.Client) *Container {
	return &Container{
		Logger: logger,
		Config: cfg,
		API:    client,
	}
}
