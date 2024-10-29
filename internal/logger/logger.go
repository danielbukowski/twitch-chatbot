package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(isDev bool) (*zap.Logger, error) {
	if !isDev {
		config := zap.NewProductionConfig()

		logger, err := config.Build()
		if err != nil {
			return nil, err
		}

		return logger, nil
	}

	err := os.MkdirAll("./tmp/logs", 0555)
	if err != nil && os.IsNotExist(err) {
		panic(err)
	}

	config := zap.NewDevelopmentConfig()
	config.OutputPaths = append(config.OutputPaths, "./tmp/logs/app.log")
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func Flush(logger *zap.Logger) {
	err := logger.Sync()
	if err != nil {
		fmt.Printf("sync method in logger threw error: %v\n", err)
	}
}
