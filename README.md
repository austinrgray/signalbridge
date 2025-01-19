# signalbridge
protoc --go_out=. --go_opt=paths=source_relative models/messages.proto

nats-server -jetstream -c test.conf

Micro Services

vessel
    status.heartbeat
    status.telemetry
    status.cargo
    status.mission
    command.mission
        assign
        update
        close
    command.telemetry
        assign
        update
    cargo
        load
        deliver

fleet
    status.heartbeat
    status.telemetry
    status.mission
    status.cargo
    dispatch.assign
    dispatch.return
    dispatch.command


cartographer
    starchart
        get.Current
        get.(Datetime)
    route_flight
        new.(FlightId)
        validate.(FlightId)
    fleet_telemetry
        get.Current
        get.(Datetime)
        update.(FlightID)



fleet

cartographer


package subjects

const (
	//mission control service
	MissionControlStream         = "mission-control-stream"
	MissionControlService        = "mission-control-svc"
	NewMissionEndpoint           = "new-mission"
	CancelMissionEndpoint        = "cancel-mission"
	HandleEmergencyEndpoint      = "handle-emergency"
	FlightsKV                    = "mission-control.flights.kv"
	MissionsKV                   = "mission-control.missions.kv"
	VesselsKV                    = "mission-control.vessels.kv"
	CartographerFlightsSentry    = "mission-control.sentry.cartographer.flights"
	CartographerCollisionsSentry = "mission-control.sentry.cartographer.collisions"
	CartographerReroutesSentry   = "mission-control.sentry.cartographer.reroutes"
	FleetMissionsSentry          = "mission-control.sentry.fleet.missions"
	FleetStatusSentry            = "mission-control.sentry.fleet.status"
	FleetTelemetrySentry         = "mission-control.sentry.fleet.telemetry"
	ManifestSentry               = "mission-control.sentry.fleet.manifests"

	//cartographer service
	CartographerStream  = "cartographer-stream"
	CartographerService = "cartographer-svc"
	CelestialsSubject   = "cartographer.celestials"
	CollisionsSubject   = "cartographer.collisions"
	FlightsSubject      = "cartographer.flights"
	ReroutesSubject     = "cartographer.reroutes"

	//fleet deck service
	FleetDeckStream    = "fleet-deck-stream"
	FleetDeckService   = "fleet-deck-service"
	ActiveVesselsKV    = "fleet-deck.vessels.active.kv"
	AvailableVesselsKV = "fleet-deck.vessels.available.kv"
	InactiveVesselsKV  = "fleet-deck.vessels.inactive.kv"
	StagedVesselsKV    = "fleet-deck.vessels.staged.kv"
	ConsignmentsKV     = "fleet-deck.consignments.kv"
	ManifestsKV        = "fleet-deck.manifests.kv"
	ReceiptsKV         = "fleet-deck.receipts.kv"

	//signal relay service
	SignalRelayStream      = "signal-relay-stream"
	SignalRelayService     = "signal-relay-svc"
	AddConsumerEndpoint    = "add-consumer"
	RemoveConsumerEndpoint = "remove-consumer"
	//relay emitter
	EmitCommandEndpoint   = "queue-emit-dispatch-commands"  //emitter.fleet.{vesselid}.command-queue
	EmitMissionEndpoint   = "queue-emit-dispatch-missions"  //emitter.fleet.{vesselid}.mission-queue
	EmitManifestEndpoint  = "queue-emit-dispatch-manifests" //emitter.fleet.{vesselid}.manifest-queue
	EmitTelemetryEndpoint = "queue-emit-dispatch-telemetry" //emitter.fleet.{vesselid}.telemetry-queue
	EmitterSubjectPrefix  = "emitter.fleet"
	//relay receiver
	FleetManifestsSubject = "receiver.fleet.*.manifests"
	FleetStatusSubject    = "receiver.fleet.*.status"
	FleetTelemetrySubject = "receiver.fleet.*.telemetry"
)

msg flows

*dispatch-web-server*   <-Pull-  "mission-control.vessels.kv"    <-  *mission-control-sentry-svc*    <-Push-  "mission-control.sentry.fleet.status"   <-  *receiver-svc*  <-Push-  "receiver.fleet.*.status"
*fleet-deck-db-svc*     <-Pull-  