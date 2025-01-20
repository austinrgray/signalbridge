package emitter

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	logging "signalbridge/0codexclone/logging"
	cfg "signalbridge/configs/emitter"

	//"github.com/austinrgray/signalcodex/logging"
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
}

func NewEmitter() *Emitter {
	return &Emitter{}
}

func (e *Emitter) InitializationSequence(ctx context.Context, cancel context.CancelCauseFunc, config cfg.Config) error {
	if err := e.Connect(ctx, cancel, config); err != nil {
		return fmt.Errorf(logging.NATSConnectFailureLogMsg+": %w", err)
	}
	if err := e.InitializeJetstream(ctx, config.JS); err != nil {
		return fmt.Errorf(logging.JetstreamInitFailureLogMsg+": %w", err)
	}
	if err := e.InitializeService(config.Micro); err != nil {
		return fmt.Errorf(logging.MicroServiceInitFailureLogMsg+": %w", err)
	}
	return nil
}

func (e *Emitter) ReInitialize(ctx context.Context, config cfg.Config) error {
	if err := e.InitializeJetstream(ctx, config.JS); err != nil {
		return fmt.Errorf(logging.JetstreamInitFailureLogMsg+": %w", err)
	}
	if err := e.InitializeService(config.Micro); err != nil {
		return fmt.Errorf(logging.MicroServiceInitFailureLogMsg+": %w", err)
	}
	return nil
}

func (e *Emitter) Start(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	config cfg.Config,
	mainWg *sync.WaitGroup) {
	defer mainWg.Done()
	startWg := sync.WaitGroup{}
	logger := LoggerFromContext(ctx)

	startWg.Add(1)
	go func() {
		<-ctx.Done()
		e.Stop(ctx)
		startWg.Done()
	}()

	if err := e.addEmitCommandEndpoint(ctx, config.Micro.Endpoints["EmitCommand"]); err != nil {
		logger.Error(logging.EndpointAddFailureLogMsg, zap.Error(err))
		cancel(fmt.Errorf(logging.NetworkErrorSignal))
	} else {
		logger.Info(logging.EndpointAddSuccessLogMsg)
	}

	startWg.Wait()
}

func (e *Emitter) Stop(ctx context.Context) {
	logger := LoggerFromContext(ctx)

	logger.Info(logging.NATSDrainingConnectionLogMsg)
	if err := e.nconn.Drain(); err != nil {
		logger.Warn(logging.NATSDrainingErrorLogMsg, zap.Error(err))
	}

	if !e.nconn.IsClosed() {
		logger.Info(logging.NATSClosingConnectionLogMsg)
		e.nconn.Close()
	}
	logger.Info(logging.ServicesStoppedLogMsg)
}

func (e *Emitter) Connect(ctx context.Context, cancel context.CancelCauseFunc, config cfg.Config) error {
	opts := append(config.NATS.Options, nats.ReconnectHandler(func(nconn *nats.Conn) {
		logger := ctx.Value(cfg.ContextLoggerKey).(*zap.Logger)

		e.nconn = nconn

		if err := e.ReInitialize(ctx, config); err != nil {
			logger.Error("failed to re-initialize connections", zap.Error(err))
			cancel(fmt.Errorf(logging.NetworkErrorSignal))
		}

		logger.Info("successfully reconnected to nats server",
			zap.String("server address", config.NATS.ServerAddr),
			zap.String("stream name", config.JS.EmitStream.Config.Name),
			zap.String("service name", config.ServiceName))
	}))

	var err error
	e.nconn, err = nats.Connect(config.NATS.ServerAddr, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) InitializeJetstream(ctx context.Context, config cfg.JSConfig) error {
	var err error
	e.js, err = jetstream.New(e.nconn, config.Options...)
	if err != nil {
		return err
	}

	ctx = NewLoggerContext(ctx, zap.String("stream", config.EmitStream.Config.Name))
	e.stream, err = e.js.CreateOrUpdateStream(ctx, config.EmitStream.Config)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) InitializeService(config cfg.MicroConfig) error {
	var err error
	e.svc, err = micro.AddService(e.nconn, config.Config)
	if err != nil {
		return err
	}
	return nil
}

func (e *Emitter) addEmitCommandEndpoint(ctx context.Context, config cfg.EndpointConfig) error {
	handler := newEmitHandler(
		NewLoggerContext(ctx, zap.String("handler", config.Name)),
		e.js,
		config.PublishOptions,
		[]micro.RespondOpt{})

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
}

func newEmitHandler(
	ctx context.Context,
	js jetstream.JetStream,
	publishOpts []jetstream.PublishOpt,
	microRespondOpts []micro.RespondOpt,
) EmitHandler {
	return EmitHandler{
		ctx:              ctx,
		js:               js,
		publishOpts:      publishOpts,
		microRespondOpts: microRespondOpts,
	}
}

func (h EmitHandler) Handle(req micro.Request) {
	switch req.Headers().Get(models.MessageTypeHeaderField) {
	case models.CommandMsgType:
		h.QueueEmitCommand(req)
	default:
		LoggerFromContext(h.ctx).Warn(fmt.Sprintf(
			"Unrecognized msg type h:%s d:%s",
			req.Headers().Get("msg_type"), string(req.Data())))
	}
}

func (h *EmitHandler) QueueEmitCommand(req micro.Request) {
	ctx, cancel := context.WithTimeout(h.ctx, time.Duration(5*time.Second))
	defer cancel()

	header := nats.Header(req.Headers())
	vesselId := header.Get("vessel-id")
	subject := fmt.Sprintf("relay.emitter.%s.commands", vesselId)
	reply := subject + ".reply"

	var logger *zap.Logger
	newSpanId, err := utils.GenerateSpanID()
	if err != nil {
		header.Set(models.SpanIdHeaderField, newSpanId)
		logger = LoggerWithHeader(header, LoggerFromContext(ctx))
		logger.Warn("error creating new span-id", zap.String("fallback-span-id", newSpanId))
	} else {
		header.Set(models.SpanIdHeaderField, newSpanId)
		logger = LoggerWithHeader(header, LoggerFromContext(ctx))
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

func NewLoggerContext(parent context.Context, fields ...zap.Field) context.Context {
	parentLogger := parent.Value(cfg.ContextLoggerKey).(*zap.Logger)
	logger := parentLogger.With(fields...)
	return context.WithValue(parent, cfg.ContextLoggerKey, logger)
}

func LoggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(cfg.ContextLoggerKey).(*zap.Logger)
	if !ok {
		log.Fatal("here")
		return zap.NewNop()
	}
	return logger
}
