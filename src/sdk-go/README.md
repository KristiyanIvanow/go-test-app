# sdk-go

Go SDK for **netfield-hub**. The package layout mirrors the structure of
`sdk-node` and `sdk-python` so the same concepts map 1:1 across languages.

## Layout

```
src/
  sdk.go                       # package documentation
  types/                       # enums, constants, regex
  models/                      # data models
  errors/                      # custom SDK error types
  logger/                      # ILogger interface + ConsoleLogger
  mqttapi/                     # MqttAPI interface for message dispatch
  mqttclient/                  # singleton MQTTManager
  containerproperties/         # reported / desired properties client
  directmethod/                # direct method invocation client
tests/
  unit-tests/
```

## Install dependencies

```bash
cd packages/sdk-go
go mod tidy
```

The SDK depends on paho.mqtt.golang (github.com/eclipse/paho.mqtt.golang).

## Quick start

```go
package main

import (
    "log"

    "github.com/KristiyanIvanow/go-test-app/src/models"
    "github.com/KristiyanIvanow/go-test-app/src/mqttclient"
    "github.com/KristiyanIvanow/go-test-app/src/types"
)

func main() {
    cfg := models.NewMqttConfigModel()
    cfg.ServerURIs = []string{"mqtt://localhost:1883"}

    client := mqttclient.GetInstance()
    if err := client.Init(cfg, "mycontainer"); err != nil {
        log.Fatal(err)
    }
    defer client.DeInit()

    if err := client.PublishMessage("hello/world", "hi from go", types.QoS1, false); err != nil {
        log.Fatal(err)
    }
}
```

## Tests

```bash
go test ./...
```
