package main

import (
	cfg "signalbridge/configs/emitter"
	relay "signalbridge/internal/emitter"

	"github.com/austinrgray/signalcodex/logging"
	"go.uber.org/zap"
	//"github.com/austinrgray/signalcodex/models"
	//"github.com/austinrgray/signalcodex/messages"
)

func main() {
	logger, err := logging.InitLogger("emitter-svc")
	if err != nil {
		logger.Warn("error initializing file-based logger: using default instead", zap.Error(err))
	}

	config, err := cfg.LoadConfig()
	if err != nil {
		logger.Warn("error loading config", zap.Error(err))
	}

	emitter := relay.NewEmitter(logger)
	emitter.Start(config)
}
