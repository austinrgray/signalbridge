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