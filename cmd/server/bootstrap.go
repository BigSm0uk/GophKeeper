package main

import (
	"github.com/BigSm0uk/GophKeeper/internal/server/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/server/app/logger"
	"go.uber.org/zap"
)

func bootstrap() error {
	// Read config
	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}
	// Init logger
	zapLogger, err := logger.NewZapLogger(cfg.Logger.Level, cfg.Env == config.EnvDevelopment)
	if err != nil {
		return err
	}
	zapLogger.Debug("Config readed", zap.Any("cfg", cfg))
	// Init db connections
	// Init repo
	// Init services
	// Init handlers

	return nil
}
