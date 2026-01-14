package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(level string, isDevelopment bool) (*zap.Logger, error) {
    var config zap.Config

    if isDevelopment {
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    } else {
        config = zap.NewProductionConfig()
    }

    var zapLevel zapcore.Level
    if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
        zapLevel = zapcore.InfoLevel
    }
    config.Level = zap.NewAtomicLevelAt(zapLevel)

    return config.Build()
}
