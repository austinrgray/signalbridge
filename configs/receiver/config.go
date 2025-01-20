package receiver_config

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go/micro"
)

type Config struct {
	NATS  NATSConfig
	JS    JSConfig
	Micro MicroConfig
}

func LoadConfig() (Config, error) {
	return Config{
		NATS:  loadNATSConfig(),
		JS:    loadJSConfig(),
		Micro: loadMicroConfig(),
	}, nil
}

type NATSConfig struct {
	ServerAddr string
	Options    []nats.Option
}

func loadNATSConfig() NATSConfig {
	return NATSConfig{
		ServerAddr: "nats://127.0.0.1:4222",
		Options: []nats.Option{
			nats.Name("receiver-service"),
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
			// nats.ReconnectHandler(),
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
			// nats.RetryOnFailedConnect(),
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
	Name     string
	Options  []jetstream.JetStreamOpt
	Streams  []StreamConfig
	KVStores []KeyValueConfig
}

func loadJSConfig() JSConfig {
	return JSConfig{
		Name:    "receiver-stream",
		Options: []jetstream.JetStreamOpt{
			// jetstream.WithClientTrace(ct),
			// jetstream.WithPublishAsyncErrHandler(cb),
			// jetstream.WithPublishAsyncMaxPending(max),
		},
		Streams: []StreamConfig{
			streamConfig(),
		},
	}
}

type StreamConfig struct {
	Config jetstream.StreamConfig
}

func streamConfig() StreamConfig {
	return StreamConfig{
		Config: jetstream.StreamConfig{
			Name:        "receiver-stream",
			Description: "", //subjects.signalrelaystreamdescription
			Subjects:    []string{"fleet.>"},
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

type KeyValueConfig struct {
	Config jetstream.KeyValueConfig
}

type MicroConfig struct {
	Config    micro.Config
	Endpoints []EndpointConfig
}

func loadMicroConfig() MicroConfig {
	return MicroConfig{
		Config: micro.Config{
			Name: "receiver-service",
			//Endpoint: ,
			Version:     "0.0.1",
			Description: "",
			//Metadata: ,
			//QueueGroup
			//StatsHandler: ,
			//DoneHandler: ,
			//ErrorHandler: ,
		},
		Endpoints: []EndpointConfig{
			emitCommandConfig(),
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
