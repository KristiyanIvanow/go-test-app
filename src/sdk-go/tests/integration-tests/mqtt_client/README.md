# Integration Tests — MQTT Client (Go)

Integration tests that exercise the netFIELD Hub Go SDK against a **live
MQTT broker**. Mirrors the `.NET` and Node integration tests.

## What's in here

- `main.go` — runnable demo program (connects, subscribes, publishes,
  starts a 1-minute topic scan, prints received messages until Ctrl+C).
  Equivalent to the .NET `Program.cs`.
- `main_test.go` — `go test` integration suite gated behind the
  `integration` build tag so it is skipped during normal `go test ./...`.

## Requirements

- A reachable MQTT broker. By default the tests target
  `mqtt://127.0.0.1:1883` (e.g. local mosquitto).
- Override with the `MQTT_URI` environment variable, e.g.
  `MQTT_URI=mqtt://mosquitto:1883` when running inside docker compose.

## Running the demo

```bash
cd packages/sdk-go
go run ./tests/integration-tests/mqtt_client
# or with a remote broker
MQTT_URI=mqtt://mosquitto:1883 go run ./tests/integration-tests/mqtt_client
```

You'll see a `Connection state` line, then any messages published to
`netfield/test/#` echoed back to the console.

To send a message from another shell:

```bash
mosquitto_pub -h localhost -t netfield/test/hello -m "hi"
```

## Running the integration tests

```bash
cd packages/sdk-go
go test -tags=integration ./tests/integration-tests/mqtt_client/...
```

The tests **skip themselves** if the broker port is not reachable, so
they are safe to include in CI without forcing a broker sidecar.

## Notes

- The default `tests/unit-tests` suite is broker-free and runs with
  plain `go test ./...`.
- The integration tests share the SDK singleton (`mqttclient.GetInstance`),
  so they are not parallel-safe — `go test` runs them sequentially.
