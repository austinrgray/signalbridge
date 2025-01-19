package receiver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfg "signalbridge/configs/receiver"

	"github.com/austinrgray/signalcodex/logging"
	"github.com/austinrgray/signalcodex/utils"

	"github.com/austinrgray/signalcodex/models"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
	"go.uber.org/zap"
)

type Receiver struct {
	nconn    *nats.Conn
	svc      micro.Service
	js       jetstream.JetStream
	stream   jetstream.Stream
	kvStores map[string]jetstream.KeyValue
	logger   *zap.Logger
}

func NewReceiver(logger *zap.Logger) *Receiver {
	return &Receiver{
		logger: logger,
	}
}

type ContextKey int

const (
	ContextLogger ContextKey = iota
)

func (r *Receiver) Start(config cfg.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan // Wait for a signal
		r.logger.Info(logging.ReceivedTerminationSignalLogMsg)
		cancel() // Cancel the context
	}()

	if err := r.connect(config.NATS); err != nil {
		r.logger.Fatal(logging.NATSConnectFailureLogMsg, zap.Error(err))
		return
	}
	r.logger.Info(logging.NATSConnectSuccessLogMsg)

	if err := r.initJetstream(ctx, config.JS); err != nil {
		r.logger.Fatal(logging.JetstreamInitFailureLogMsg, zap.Error(err))
		return
	}
	r.logger.Info(logging.JetstreamInitSuccessLogMsg)

	if err := r.subscribeFleetStatus(ctx); err != nil {
		r.logger.Fatal("failed to initialize subscription", zap.Error(err))
		return
	}
	r.logger.Info(logging.JetstreamInitSuccessLogMsg)

	<-ctx.Done()
	r.Stop()
}

func (r *Receiver) Stop() {
	r.logger.Info(logging.DrainingNATSConnectionLogMsg)
	if err := r.nconn.Drain(); err != nil {
		r.logger.Warn(logging.NATSDrainingErrorLogMsg, zap.Error(err))
	}
	r.logger.Info(logging.ServiceShutdownLogMsg)
}

func (r *Receiver) connect(config cfg.NATSConfig) error {
	var err error
	r.nconn, err = nats.Connect(config.ServerAddr, config.Options...)
	if err != nil {
		return err
	}
	return nil
}

func (r *Receiver) initJetstream(ctx context.Context, config cfg.JSConfig) error {
	var err error
	r.js, err = jetstream.New(r.nconn, config.Options...)
	if err != nil {
		return err
	}
	logger := r.logger.With(zap.String("stream", config.Name))
	ctx = context.WithValue(ctx, ContextLogger, logger)
	r.stream, err = r.js.CreateOrUpdateStream(ctx, config.Streams[0].Config)
	if err != nil {
		return err
	}
	return nil
}

// func (r *Receiver) initKVStore(ctx context.Context, key string, config jetstream.KeyValueConfig) error {
// 	var err error
// 	r.kvStores[key], err = r.js.CreateOrUpdateKeyValue(ctx, config)
// 	if err != nil {
// 		return err
// 	}
// }

func (r *Receiver) subscribeFleetStatus(ctx context.Context) error {
	handler := newReceiveHandler(ctx, r.js, r.logger)
	_, err := r.nconn.Subscribe("relay.fleet.*.status", handler.HandleFleetStatus())
	if err != nil {
		return err
	}
	return nil
}

func (h *ReceiveHandler) HandleFleetStatus() nats.MsgHandler {
	return func(msg *nats.Msg) {
		ctx, cancel := context.WithTimeout(h.ctx, time.Duration(10*time.Second))
		defer cancel()

		header := nats.Header(msg.Header)
		subject := "mission_control.sentry.fleet.status"
		reply := subject + ".reply"

		var logger *zap.Logger
		newSpanId, err := utils.GenerateSpanID()
		if err != nil {
			header.Set(models.SpanIdHeaderField, newSpanId)
			logger = LoggerWithHeader(header, h.logger)
			logger.Warn("error creating new span-id", zap.String("fallback-span-id", newSpanId))
		} else {
			header.Set(models.SpanIdHeaderField, newSpanId)
			logger = LoggerWithHeader(header, h.logger)
		}

		newMsg := NewNatsMessage(subject, reply, header, msg.Data)

		ack, err := h.js.PublishMsg(ctx, newMsg, h.publishOpts...)
		if err != nil {
			description := fmt.Sprintf("%s to %s", "failed to publish", newMsg.Subject)
			logger.Error(description, zap.Error(err))
		}
		logger.Info(logging.EmitHandlerPublishSuccessLogMsg, zap.Uint("ack-sequence", uint(ack.Sequence)))
	}
}

type ReceiveHandler struct {
	ctx              context.Context
	js               jetstream.JetStream
	kv               jetstream.KeyValue
	publishOpts      []jetstream.PublishOpt
	microRespondOpts []micro.RespondOpt
	logger           *zap.Logger
}

func newReceiveHandler(
	ctx context.Context,
	js jetstream.JetStream,
	// publishOpts []jetstream.PublishOpt,
	// microRespondOpts []micro.RespondOpt,
	logger *zap.Logger,
) ReceiveHandler {
	return ReceiveHandler{
		ctx: ctx,
		js:  js,
		// publishOpts:      publishOpts,
		// microRespondOpts: microRespondOpts,
		logger: logger,
	}
}

func (h ReceiveHandler) Handle(req micro.Request) {
	switch req.Headers().Get(models.MessageTypeHeaderField) {
	case models.CommandMsgType:
		//h.QueueEmitCommand(req)
	default:
		h.logger.Error(fmt.Sprintf("Unrecognized msg type h:%s d:%s", req.Headers().Get("msg-type"), string(req.Data())))
	}
}

func NewNatsMessage(subject string, reply string, header nats.Header, data []byte) *nats.Msg {
	return &nats.Msg{
		Subject: subject,
		Reply:   reply,
		Header:  header,
		Data:    data,
	}
}

func LoggerWithHeader(header nats.Header, logger *zap.Logger) *zap.Logger {
	fields := make([]zap.Field, 0)
	for k, v := range header {
		if len(v) == 1 {
			fields = append(fields, zap.String(k, v[0]))
			continue
		}
		for i, val := range v {
			fields = append(fields, zap.String(fmt.Sprintf("%s[%d]", k, i), val))
		}
	}
	return logger.With(fields...)
}
