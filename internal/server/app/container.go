package app

import (
	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"go.uber.org/zap"
)

type Server struct {
	logger *zap.Logger
	cfg *config.ServerConfig
}
