package main

import (
	"context"
	"jfrog-assignment/cmd"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Custom console encoder with colors
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Create context that can be canceled on signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the application
	doneChan := make(chan struct{})
	go func() {
		cmd.Execute(ctx, logger)
		close(doneChan)
	}()

	// Wait for completion or signal
	select {
	case <-doneChan:
		logger.Info("application completed normally")
	case sig := <-sigChan:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel() // Cancel the context to start shutdown

		// Give goroutines 5 seconds to finish
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		select {
		case <-doneChan:
			logger.Info("application shutdown gracefully")
		case <-shutdownCtx.Done():
			logger.Warn("shutdown timeout exceeded, forcing exit",
				zap.Duration("timeout", 5*time.Second))
		}
	}
}
