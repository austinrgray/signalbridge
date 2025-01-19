package main

import (
	"context"
	"fmt"
	"time"

	"github.com/austinrgray/signalcodex/logging"
	"github.com/austinrgray/signalcodex/models"
	"github.com/austinrgray/signalcodex/utils"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

func main() {
	logger, err := logging.InitLogger("mission-control-svc")
	if err != nil {
		logger.Warn("error initializing file-based logger: using default instead", zap.Error(err))
	}

	nc, err := nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		logger.Fatal(logging.NATSConnectFailureLogMsg, zap.Error(err))
		return
	}

	js, err := jetstream.New(nc)
	if err != nil {
		logger.Fatal(logging.JetstreamInitFailureLogMsg, zap.Error(err))
		return
	}

	ctx := context.Context(context.Background())
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "mission-control-stream",
		Description: "",
		Subjects:    []string{"mission_control.>"},
		Retention:   jetstream.WorkQueuePolicy,
		Discard:     jetstream.DiscardOld,
		Storage:     jetstream.MemoryStorage,
	})
	if err != nil {
		logger.Fatal(logging.JetstreamInitFailureLogMsg, zap.Error(err))
		return
	}
	_ = stream

	consumer, err := js.CreateOrUpdateConsumer(ctx, "mission-control-stream", jetstream.ConsumerConfig{
		Durable:       "mc-sentry-status",
		Description:   "",
		FilterSubject: "mission_control.sentry.fleet.status",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		logger.Fatal("failed to initialize consumer", zap.Error(err))
		return
	}

	handler := func(msg jetstream.Msg) {
		logger.Info("consumed message", zap.String("data", string(msg.Data())))

		header := msg.Headers()
		header.Set("msg_type", "command")
		subject := "queue-emit-command-request"
		reply := subject + ".reply"

		var nlogger *zap.Logger
		newSpanId, err := utils.GenerateSpanID()
		if err != nil {
			header.Set(models.SpanIdHeaderField, newSpanId)
			nlogger = LoggerWithHeader(header, logger)
			nlogger.Warn("error creating new span-id", zap.String("fallback-span-id", newSpanId))
		} else {
			header.Set(models.SpanIdHeaderField, newSpanId)
			nlogger = LoggerWithHeader(header, logger)
		}

		newMsg := NewNatsMessage(subject, reply, header, msg.Data())

		res, err := nc.RequestMsg(newMsg, time.Duration(10*time.Second))
		if err != nil {
			description := fmt.Sprintf("%s to %s", "request failed", newMsg.Subject)
			nlogger.Error(description, zap.Error(err))
		}
		_ = res
		nlogger.Info("response", zap.String("res", string(res.Data)))
		msg.Ack()
	}

	for {
		consumer.Consume(handler, []jetstream.PullConsumeOpt{}...)
	}

	// kv, err := js.CreateOrUpdateKeyValue(context.Context(context.Background()), jetstream.KeyValueConfig{
	// 	Bucket:      "mission_control.fleet.*.status.kv",
	// 	Description: "bucket of latest status from active fleet",
	// 	History:     2,
	// 	Replicas:    0,
	// })
	// if err != nil {
	// 	logger.Fatal("error initializing status kv store", zap.Error(err))
	// }
	// logger.Info("FLEET_STATUS initialized")

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
