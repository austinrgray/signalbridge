package emitter_config

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
)

const ServiceName = "relay-emitter"

type ContextKey int

const (
	ContextLoggerKey ContextKey = iota
)

type Config struct {
	ServiceName       string
	NATS              NATSConfig
	JS                JSConfig
	Micro             MicroConfig
	MaxInitRetries    int
	InitRetryDelay    time.Duration
	InitResetInterval time.Duration
}

func LoadConfig() (Config, error) {
	return Config{
		ServiceName:       ServiceName,
		NATS:              loadNATSConfig(),
		JS:                loadJSConfig(),
		Micro:             loadMicroConfig(),
		MaxInitRetries:    15,
		InitRetryDelay:    15 * time.Second,
		InitResetInterval: 10 * time.Minute,
	}, nil
}

type NATSConfig struct {
	ServerAddr string
	Options    []nats.Option
}

type HandlerKey int

const (
	NATSReconnectHandlerKey HandlerKey = iota
)

func loadNATSConfig() NATSConfig {
	return NATSConfig{
		ServerAddr: "nats://127.0.0.1:4222",
		Options: []nats.Option{
			nats.Name(ServiceName),
			// nats.InProcessServer(),
			// nats.Secure(),
			// nats.ClientTLSConfig(),
			// nats.RootCAs(),
			// nats.ClientCert(),
			// nats.NoReconnect(),
			// nats.DontRandomize(),
			// nats.NoEcho(),
			// nats.ReconnectWait(),
			// nats.MaxReconnects(),
			// nats.ReconnectJitter(),
			// nats.CustomReconnectDelay(),
			// nats.PingInterval(),
			// nats.MaxPingsOutstanding(),
			// nats.ReconnectBufSize(),
			// nats.Timeout(),
			// nats.FlusherTimeout(),
			// nats.DrainTimeout(),
			// nats.DisconnectErrHandler(),
			// nats.DisconnectHandler(),
			// nats.ConnectHandler(),
			//nats.ReconnectHandler(),
			// nats.ClosedHandler(),
			// nats.DiscoveredServersHandler(),
			// nats.ErrorHandler(),
			// nats.ErrorHandler(),
			// nats.UserInfoHandler(),
			// nats.Token(),
			// nats.TokenHandler(),
			// nats.UserCredentials(),
			// nats.UserJWTAndSeed(),
			// nats.UserJWT(),
			// nats.Nkey(),
			// nats.SyncQueueLen(),
			// nats.Dialer(),
			// nats.SetCustomDialer(),
			// nats.UseOldRequestStyle(),
			// nats.NoCallbacksAfterClientClose(),
			// nats.LameDuckModeHandler(),
			nats.RetryOnFailedConnect(true),
			// nats.Compression(),
			// nats.ProxyPath(),
			// nats.CustomInboxPrefix(),
			// nats.IgnoreAuthErrorAbort(),
			// nats.SkipHostLookup(),
			// nats.PermissionErrOnSubscribe(),
			// nats.TLSHandshakeFirst(),
		},
	}
}

type JSConfig struct {
	Options    []jetstream.JetStreamOpt
	EmitStream StreamConfig
}

func loadJSConfig() JSConfig {
	return JSConfig{
		Options: []jetstream.JetStreamOpt{
			// jetstream.WithClientTrace(ct),
			// jetstream.WithPublishAsyncErrHandler(cb),
			// jetstream.WithPublishAsyncMaxPending(max),
		},
		EmitStream: emitStreamConfig(),
	}
}

type StreamConfig struct {
	Config jetstream.StreamConfig
}

func emitStreamConfig() StreamConfig {
	return StreamConfig{
		Config: jetstream.StreamConfig{
			Name:        ServiceName + "-stream",
			Description: "", //subjects.signalrelaystreamdescription
			Subjects:    []string{"relay.emitter.>"},
			Retention:   jetstream.WorkQueuePolicy,
			//MaxConsumers: ,
			//MaxMsgs: ,
			//MaxBytes: ,
			Discard: jetstream.DiscardOld,
			//DiscardNewPerSubject: ,
			// MaxAge time.Duration
			// MaxMsgsPerSubject int64
			// MaxMsgSize int32
			Storage: jetstream.FileStorage,
			// Replicas int
			// NoAck bool
			// Duplicates time.Duration
			// Placement *Placement
			// Mirror *StreamSource
			// Sources []*StreamSource
			// Sealed bool
			// DenyDelete bool
			// DenyPurge bool
			// AllowRollup bool
			// Compression StoreCompression
			// FirstSeq uint64
			// SubjectTransform *SubjectTransformConfig
			// RePublish *RePublish
			// AllowDirect bool
			// MirrorDirect bool
			// ConsumerLimits StreamConsumerLimits
			// Metadata map[string]string
			// Template string
		},
	}
}

type MicroConfig struct {
	Config    micro.Config
	Endpoints map[string]EndpointConfig
}

func loadMicroConfig() MicroConfig {
	return MicroConfig{
		Config: micro.Config{
			Name: ServiceName + "-service",
			//Endpoint: ,
			Version:     "0.0.1",
			Description: "",
			//Metadata: ,
			//QueueGroup
			//StatsHandler: ,
			//DoneHandler: ,
			//ErrorHandler: ,
		},
		Endpoints: map[string]EndpointConfig{
			"EmitCommand": emitCommandConfig(),
		},
	}
}

type EndpointConfig struct {
	Name           string
	Options        []micro.EndpointOpt
	PublishOptions []jetstream.PublishOpt
}

func emitCommandConfig() EndpointConfig {
	return EndpointConfig{
		Name: "queue-emit-command-request",
		//Handler: ,
		Options: []micro.EndpointOpt{
			micro.WithEndpointMetadata(map[string]string{
				"Description":     "",
				"format":          "application/protobuf",
				"request_schema":  "proto.DispatchCommand",
				"response_schema": "proto.DispatchCommandAck",
			}),
			// micro.WithEndpointMetadata(),
			// micro.WithEndpointSubject(),
		},
		//QueueGroup: ,
		PublishOptions: []jetstream.PublishOpt{
			// jetstream.WithExpectLastMsgID(),
			// jetstream.WithExpectLastSequence(),
			// jetstream.WithExpectLastSequence(),
			// jetstream.WithExpectStream(),
			// jetstream.WithMsgID(),
			// jetstream.WithRetryAttempts(),
			// jetstream.WithRetryWait(),
			// jetstream.WithStallWait(),
		},
	}
}
