package emitter

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfg "signalbridge/configs/emitter"

	"github.com/austinrgray/signalcodex/logging"
	"github.com/austinrgray/signalcodex/utils"

	"github.com/austinrgray/signalcodex/models"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
	"go.uber.org/zap"
)

type Emitter struct {
	nconn  *nats.Conn
	svc    micro.Service
	js     jetstream.JetStream
	stream jetstream.Stream
	logger *zap.Logger
}

func NewEmitter(logger *zap.Logger) *Emitter {
	return &Emitter{
		logger: logger,
	}
}

type ContextKey int

const (
	ContextLogger ContextKey = iota
)

func (e *Emitter) Start(config cfg.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan // Wait for a signal
		e.logger.Info(logging.ReceivedTerminationSignalLogMsg)
		cancel() // Cancel the context
	}()

	if err := e.connect(config.NATS); err != nil {
		e.logger.Fatal(logging.NATSConnectFailureLogMsg, zap.Error(err))
		return
	}
	e.logger.Info(logging.NATSConnectSuccessLogMsg)

	if err := e.initJetstream(ctx, config.JS); err != nil {
		e.logger.Fatal(logging.JetstreamInitFailureLogMsg, zap.Error(err))
		return
	}
	e.logger.Info(logging.JetstreamInitSuccessLogMsg)

	if err := e.initService(config.Micro); err != nil {
		e.logger.Fatal(logging.MicroServiceInitFailureLogMsg, zap.Error(err))
		return
	}
	e.logger.Info(logging.MicroServiceInitSuccessLogMsg)

	if err := e.addEmitCommandEndpoint(ctx, e.logger, config.Micro.Endpoints[0]); err != nil {
		e.logger.Fatal(logging.EndpointAddFailureLogMsg, zap.Error(err))
		return
	}
	e.logger.Info(logging.EndpointAddSuccessLogMsg)

	<-ctx.Done()
	e.Stop()
}

func (e *Emitter) Stop() {
	e.logger.Info(logging.DrainingNATSConnectionLogMsg)
	if err := e.nconn.Drain(); err != nil {
		e.logger.Warn(logging.NATSDrainingErrorLogMsg, zap.Error(err))
	}
	e.logger.Info(logging.ServiceShutdownLogMsg)
}

func (e *Emitter) connect(config cfg.NATSConfig) error {
	var err error
	e.nconn, err = nats.Connect(config.ServerAddr, config.Options...)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) initJetstream(ctx context.Context, config cfg.JSConfig) error {
	var err error
	e.js, err = jetstream.New(e.nconn, config.Options...)
	if err != nil {
		return err
	}
	logger := e.logger.With(zap.String("stream", config.Name))
	ctx = context.WithValue(ctx, ContextLogger, logger)
	e.stream, err = e.js.CreateOrUpdateStream(ctx, config.Streams[0].Config)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) initService(config cfg.MicroConfig) error {
	var err error
	e.svc, err = micro.AddService(e.nconn, config.Config)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) addEmitCommandEndpoint(ctx context.Context, logger *zap.Logger, config cfg.EndpointConfig) error {
	logger = logger.With(zap.String("handler", config.Name))
	ctx = context.WithValue(ctx, ContextLogger, logger)

	handler := newEmitHandler(ctx, e.js, config.PublishOptions, []micro.RespondOpt{}, logger)

	err := e.svc.AddEndpoint(config.Name, handler, config.Options...)
	if err != nil {
		return err
	}
	return nil
}

type EmitHandler struct {
	ctx              context.Context
	js               jetstream.JetStream
	publishOpts      []jetstream.PublishOpt
	microRespondOpts []micro.RespondOpt
	logger           *zap.Logger
}

func newEmitHandler(
	ctx context.Context,
	js jetstream.JetStream,
	publishOpts []jetstream.PublishOpt,
	microRespondOpts []micro.RespondOpt,
	logger *zap.Logger,
) EmitHandler {
	return EmitHandler{
		ctx:              ctx,
		js:               js,
		publishOpts:      publishOpts,
		microRespondOpts: microRespondOpts,
		logger:           logger,
	}
}

func (h EmitHandler) Handle(req micro.Request) {
	switch req.Headers().Get(models.MessageTypeHeaderField) {
	case models.CommandMsgType:
		h.QueueEmitCommand(req)
	default:
		h.logger.Error(fmt.Sprintf("Unrecognized msg type h:%s d:%s", req.Headers().Get("msg_type"), string(req.Data())))
	}
}

func (h *EmitHandler) QueueEmitCommand(req micro.Request) {
	ctx, cancel := context.WithTimeout(h.ctx, time.Duration(5*time.Second))
	defer cancel()

	header := nats.Header(req.Headers())
	vesselId := header.Get("vessel-id")
	subject := "fleet." + vesselId + ".dispatch-commands"
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

	msg := NewNatsMessage(subject, reply, header, req.Data())

	ack, err := h.js.PublishMsg(ctx, msg, h.publishOpts...)
	if err != nil {
		description := fmt.Sprintf("%s to %s", logging.EmitHandlerPublishFailedLogMsg, msg.Subject)
		req.Error("PublishFailed", description, []byte(err.Error()), h.microRespondOpts...)
		logger.Error(description, zap.Error(err))
	}
	logger.Info(logging.EmitHandlerPublishSuccessLogMsg, zap.Uint("ack-sequence", uint(ack.Sequence)))
	req.Respond([]byte("ok"), h.microRespondOpts...)
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
