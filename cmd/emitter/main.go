package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	logging "signalbridge/0codexclone/logging"
	cfg "signalbridge/configs/emitter"

	//"github.com/austinrgray/signalcodex/logging"
	internal "signalbridge/internal/emitter"
	"sync"
	"syscall"

	//"github.com/austinrgray/signalcodex/models"
	//"github.com/austinrgray/signalcodex/messages"

	"go.uber.org/zap"
)

func main() {
	logger, err := logging.InitLogger(cfg.ServiceName)
	if err != nil {
		logger.Warn("error initializing file-based logger",
			zap.String("resolution", "using default instead"),
			zap.Error(err))
	}

	config, err := cfg.LoadConfig()
	if err != nil {
		logger.Warn("error loading config",
			zap.String("resolution", "using default instead"),
			zap.Error(err))
	}

	ctx, cancel := context.WithCancelCause(context.WithValue(
		context.Background(), cfg.ContextLoggerKey, logger))

	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		logger.Info(logging.ReceivedTerminationSignalLogMsg)
		cancel(fmt.Errorf(logging.OSTermSignal))
	}()
	var mainWg = sync.WaitGroup{}
	var lastAttempt time.Time

	retryCount := 0
	for retryCount <= config.MaxInitRetries {
		connCtx, connCancel := context.WithCancelCause(ctx)
		defer cancel(nil)

		emitter := internal.NewEmitter()

		lastAttempt = time.Now()
		if err := emitter.InitializationSequence(connCtx, connCancel, config); err != nil {
			logger.Error("error initializing connections",
				zap.String("server address", config.NATS.ServerAddr),
				zap.String("stream name", config.JS.EmitStream.Config.Name),
				zap.String("service name", config.ServiceName),
				zap.Error(err))
			connCancel(fmt.Errorf(logging.NetworkErrorSignal))
		} else {
			logger.Info("successfully intialized connections",
				zap.String("server address", config.NATS.ServerAddr),
				zap.String("stream name", config.JS.EmitStream.Config.Name),
				zap.String("service name", config.ServiceName))
			mainWg.Add(1)
			go emitter.Start(connCtx, connCancel, config, &mainWg)
		}

		<-connCtx.Done()
		cause := context.Cause(connCtx)
		if !isRecoverable(cause) {
			logger.Error("unrecoverable error", zap.Error(err))
			break
		}
		if retryCount > config.MaxInitRetries {
			if time.Since(lastAttempt) < config.InitResetInterval {
				logger.Info("retry limit exceeded")
				break
			}
			retryCount = 0
			logger.Info("retry count reset due to timeout")
		}
		retryCount++
		logger.Info(logging.RestartServiceConnectionsLogMsg, zap.Uint(
			"retry count", uint(retryCount)))
		time.Sleep(config.InitRetryDelay)
	}
	mainWg.Wait()
	logger.Info(logging.ServiceShutdownLogMsg)
}

func isRecoverable(cause error) bool {
	switch cause.Error() {
	case logging.NetworkErrorSignal:
		return true
	case logging.TransientFailureSignal:
		return true
	default:
		return false
	}
}
