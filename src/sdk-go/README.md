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

    "netfield-sdk-go/src/models"
    "netfield-sdk-go/src/mqttclient"
    "netfield-sdk-go/src/types"
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


## Direct Method Usage

### Receiving Direct Methods (Server/Module Side)

```go
import (
  "netfield-sdk-go/src/directmethod"
  "netfield-sdk-go/src/logger"
  "netfield-sdk-go/src/models"
)

func main() {
  log := logger.NewConsoleLogger(logger.Info)
  handlers := map[string]directmethod.DirectMethodHandler{
    "GetStatus": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
      return directmethod.DirectMethodResponse{
        Status: 200,
        Payload: map[string]interface{}{"status": "running"},
      }, nil
    },
  }
  client, _ := directmethod.GetInstance("myModule", handlers)
  config := models.NewMqttConfigModel()
  config.ServerURIs = []string{"mqtt://localhost:1883"}
  client.Init(config, log)
  defer client.Stop()
  // App runs, handlers are called automatically
}
```

### Invoking Direct Methods (Client Side)

The SDK does not provide a built-in invoker, but you can invoke direct methods using any MQTT client:

```go
import (
  "encoding/json"
  "fmt"
  mqtt "github.com/eclipse/paho.mqtt.golang"
  "time"
)

type DirectMethodRequest struct {
  MessageID     string `json:"messageId"`
  ContainerName string `json:"containerName"`
  MethodName    string `json:"methodName"`
  Payload       string `json:"payload"`
}

func invokeDirectMethod(client mqtt.Client, moduleID, method string, payload interface{}) {
  msgID := fmt.Sprintf("dm-req-%d", time.Now().UnixNano())
  req := DirectMethodRequest{
    MessageID:     msgID,
    ContainerName: moduleID,
    MethodName:    method,
    Payload:       marshalPayload(payload),
  }
  body, _ := json.Marshal(req)
  topic := fmt.Sprintf("dm/%s/", moduleID)
  client.Publish(topic, 1, false, body)
  // Listen for response on dm/response/{msgID}
}

func marshalPayload(payload interface{}) string {
  b, _ := json.Marshal(payload)
  return string(b)
}
```

## Using sdk-go from Artifactory (Private Repo)

If your Go module is published to a private Artifactory Go repository:

1. Set your GOPROXY to your Artifactory Go proxy URL:
   ```bash
   export GOPROXY=https://your-artifactory-domain/artifactory/api/go/your-repo
   ```
2. Add your credentials (token or API key) to `~/.netrc`:
   ```
   machine your-artifactory-domain
   login <username or apikey>
   password <your-access-token>
   ```
3. (Optional) Set GOPRIVATE if your repo is private:
   ```bash
   export GOPRIVATE=your-artifactory-domain
   ```
4. Use `go get` as usual:
   ```bash
   go get your-artifactory-domain/your/module/path@vX.Y.Z
   ```
5. Import and use in your code:
   ```go
   import "your-artifactory-domain/your/module/path/src/directmethod"
   ```

This allows you to use sdk-go in any Go app, even if your Artifactory requires authentication.
