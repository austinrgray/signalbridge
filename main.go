package main

import (
	"context"

	"sync"

	"github.com/austinrgray/signalcodex/utils"
	//"github.com/austinrgray/signalcodex/models"
	//"github.com/austinrgray/signalcodex/messages"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
	"go.uber.org/zap"
)

const (
	FleetCargoSubject              = "fleet.*.vessel.cargo"
	FleetHeartbeatSubject          = "fleet.*.vessel.heartbeat"
	FleetTelemetrySubject          = "fleet.*.vessel.telemetry"
	CargoStatusKVStore             = "CARGO_STATUS"
	FleetStatusKVStore             = "FLEET_STATUS"
	FleetTelemetryKVStore          = "FLEET_TELEMETRY"
	SignalRelayStream              = "SIGNAL_RELAY"
	SignalBridgeStream             = "SIGNAL_BRIDGE"
	EmitterService                 = "EMITTER_SVC"
	EmitCommandEndpoint            = "DISPATCH_COMMAND_MSG"
	EmitTelemetryEndpoint          = "DISPATCH_TELEMETRY_MSG"
	EmitCargoEndpoint              = "DISPATCH_CARGO_MSG"
	RelayStreamAddConsumerEndpoint = "RELAY_STREAM_ADD_CONSUMER"
)

func main() {
	logger, err := utils.InitLogger("signalrelay")
	if err != nil {
		logger.Warn("error initializing file-based logger: using default instead", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nc, err := nats.Connect("nats://127.0.0.1:4222", []nats.Option{
		nats.Name("signal-relay"),
	}...)
	if err != nil {
		logger.Fatal("error connecting to NATS", zap.Error(err))
	}
	defer nc.Drain()
	logger.Info("connected to nats server")

	js, err := jetstream.New(nc)
	if err != nil {
		logger.Fatal("error initializing Jetstream", zap.Error(err))
	}
	logger.Info("initialized nats jetstream")

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "SIGNAL_RELAY",
		Description: "stream of long distance comms with remote vessels",
		Subjects:    []string{"fleet.*.>"},
		Retention:   jetstream.WorkQueuePolicy,
		Discard:     jetstream.DiscardOld,
		Storage:     jetstream.MemoryStorage,
	})
	if err != nil {
		logger.Fatal("error updating stream", zap.Error(err))
	}
	logger.Info("SIGNAL_RELAY initialized")

	statusKV, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      FleetStatusKVStore,
		Description: "bucket of latest heartbeats from active fleet",
		History:     2,
		Replicas:    0,
	})
	if err != nil {
		logger.Fatal("error updating status kv store", zap.Error(err))
	}
	logger.Info("FLEET_STATUS initialized")

	cargoKV, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      CargoStatusKVStore,
		Description: "bucket of pending shipments and their status",
		History:     5,
		Replicas:    1,
	})
	if err != nil {
		logger.Fatal("error updating cargo kv store", zap.Error(err))
	}
	logger.Info("CARGO_STATUS initialized")

	telemetryKV, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      FleetTelemetryKVStore,
		Description: "bucket of latest telemetry data from active fleet",
		History:     5,
		Replicas:    0,
	})
	if err != nil {
		logger.Fatal("error updating cargo kv store", zap.Error(err))
	}
	logger.Info("FLEET_TELEMETRY initialized")

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go startEmitterSvc(nc, stream, ctx, cancel, logger, wg)
	go startReceiverSvc(nc, statusKV, cargoKV, telemetryKV, ctx, cancel, logger, wg)
	wg.Wait()
}

func startReceiverSvc(
	nc *nats.Conn,
	statusKV jetstream.KeyValue,
	cargoKV jetstream.KeyValue,
	telemetryKV jetstream.KeyValue,
	ctx context.Context,
	cancel context.CancelFunc,
	logger *zap.Logger,
	pWg *sync.WaitGroup) {

	defer pWg.Done()
	defer cancel()

	logger = logger.With(zap.String("svc", "signal-reciever"), zap.String("version", "0.0.1"))

	hbSub, err := nc.Subscribe(FleetHeartbeatSubject, func(msg *nats.Msg) {
		_, err := statusKV.Put(ctx, msg.Header.Get("VesselId"), msg.Data)
		if err != nil {
			logger.Error("failed to forward heartbeat to FLEET_STATUS", zap.Error(err))
		}
	})
	if err != nil {
		logger.Fatal("failed to subscribe to fleet.*.vessel.heartbeat", zap.Error(err))
	}
	defer hbSub.Drain()
	logger.Info("subscribed to fleet heartbeats")

	cargoSub, err := nc.Subscribe(FleetCargoSubject, func(msg *nats.Msg) {
		_, err := cargoKV.Put(ctx, msg.Header.Get("ShipmentId"), msg.Data)
		if err != nil {
			logger.Error("failed to forward heartbeat to CARGO_STATUS", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("failed to subscribe to fleet.*.vessel.cargo", zap.Error(err))
	}
	defer cargoSub.Drain()
	logger.Info("subscribed to fleet cargo")

	telSub, err := nc.Subscribe(FleetTelemetrySubject, func(msg *nats.Msg) {
		_, err := telemetryKV.Put(ctx, msg.Header.Get("VesselId"), msg.Data)
		if err != nil {
			logger.Error("failed to forward heartbeat to FLEET_TELEMETRY", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("failed to subscribe to fleet.*.vessel.cargo", zap.Error(err))
		return
	}
	defer telSub.Drain()
	logger.Info("subscribed to fleet telemetry")

	select {}
}

func startEmitterSvc(
	nc *nats.Conn,
	relay jetstream.Stream, //will update consumers here
	ctx context.Context,
	cancel context.CancelFunc,
	logger *zap.Logger,
	pWg *sync.WaitGroup) {

	defer pWg.Done()
	defer cancel()
	_ = relay

	logger = logger.With(zap.String("svc", "signal-emitter"), zap.String("version", "0.0.1"))

	js, err := jetstream.New(nc)
	if err != nil {
		logger.Fatal("error initializing Jetstream", zap.Error(err))
	}

	svc, err := micro.AddService(nc, micro.Config{
		Name:        EmitterService,
		Version:     "0.0.1",
		Description: "",
	})
	if err != nil {
		logger.Fatal("failed to start EmitterSVC", zap.Error(err))
	}
	logger.Info("starting emitter service")

	err = svc.AddEndpoint(
		EmitCommandEndpoint,
		micro.HandlerFunc(func(req micro.Request) {
			vesselId := req.Headers().Get("VesselID")
			subject := "fleet." + vesselId + ".dispatch.command"

			ack, err := js.Publish(ctx, subject, req.Data(), []jetstream.PublishOpt{}...)
			if err != nil {
				logger.Error("failed to publish msg to dispatch command stream", zap.Error(err))
				req.Respond([]byte("Fail"))
			}
			_ = ack
			req.Respond([]byte("Pass"))
		}),
		micro.WithEndpointMetadata(map[string]string{
			"Description":     "",
			"format":          "application/protobuf",
			"request_schema":  "proto.DispatchCommand",
			"response_schema": "proto.DispatchCommandAck",
		}),
	)
	if err != nil {
		logger.Fatal("failed to add endpoint", zap.Error(err))
	}
	logger.Info("emit dispatch command started")

	err = svc.AddEndpoint(
		EmitTelemetryEndpoint,
		micro.HandlerFunc(func(req micro.Request) {

		}),
		micro.WithEndpointMetadata(map[string]string{
			"Description":     "",
			"format":          "application/protobuf",
			"request_schema":  "proto.DispatchCommand",
			"response_schema": "proto.DispatchCommandAck",
		}),
	)
	if err != nil {
		logger.Fatal("failed to add endpoint", zap.Error(err))
	}
	logger.Info("emit dispatch telemetry started")

	err = svc.AddEndpoint(
		EmitCargoEndpoint,
		micro.HandlerFunc(func(req micro.Request) {

		}),
		micro.WithEndpointMetadata(map[string]string{
			"Description":     "",
			"format":          "application/protobuf",
			"request_schema":  "proto.DispatchCommand",
			"response_schema": "proto.DispatchCommandAck",
		}),
	)
	if err != nil {
		logger.Fatal("failed to add endpoint", zap.Error(err))
	}
	logger.Info("emit dispatch cargo started")
}
