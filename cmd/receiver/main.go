package main

import (
	cfg "signalbridge/configs/receiver"
	relay "signalbridge/internal/receiver"

	"github.com/austinrgray/signalcodex/logging"
	"go.uber.org/zap"
	//"github.com/austinrgray/signalcodex/models"
	//"github.com/austinrgray/signalcodex/messages"
)

func main() {
	logger, err := logging.InitLogger("receiver-svc")
	if err != nil {
		logger.Warn("error initializing file-based logger: using default instead", zap.Error(err))
	}

	config, err := cfg.LoadConfig()
	if err != nil {
		logger.Warn("error loading config", zap.Error(err))
	}

	emitter := relay.NewReceiver(logger)
	emitter.Start(config)
}
