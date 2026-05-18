// Integration test demo for the MQTT client. Connects to a real broker,
// subscribes to a wildcard topic, publishes a few messages, runs a short
// scan, and exits. Mirrors tests/integration-tests/MqttClient/Program.cs
// from the .NET SDK.
//
// Run:
//
//   # against a broker on localhost:
//   go run ./tests/integration-tests/mqtt_client
//
//   # or override the broker URI:
//   MQTT_URI=mqtt://mosquitto:1883 go run ./tests/integration-tests/mqtt_client
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/mqttclient"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/types"
)

// testAPI prints every message the broker pushes to subscribed topics.
type testAPI struct{}

func (testAPI) HandleMessage(m *models.MqttMessage) {
	fmt.Println("=== Message Received ===")
	fmt.Printf("Topic:    %s\n", m.Topic)
	fmt.Printf("Payload:  %s\n", string(m.Payload))
	fmt.Printf("QoS:      %d\n", m.QoS)
	fmt.Printf("Retained: %v\n", m.Retained)
	fmt.Println("========================")
}

func main() {
	uri := os.Getenv("MQTT_URI")
	if uri == "" {
		uri = "mqtt://127.0.0.1:1883"
	}

	cfg := models.NewMqttConfigModel()
	cfg.ServerURIs = []string{uri}
	cfg.MqttVersion = 4
	cfg.ConnectTimeout = 30
	cfg.KeepAliveInterval = 60

	client := mqttclient.GetInstance()
	client.MyAPI = testAPI{}

	log.Printf("Connecting to %s ...", uri)
	if err := client.Init(cfg, "TestApp123"); err != nil {
		log.Fatalf("failed to init MQTT client: %v", err)
	}
	defer client.DeInit()

	state := client.GetConnState()
	log.Printf("Connection state: %s — %s", state.ConnectionState, state.Message)

	if !client.IsConnected() {
		log.Fatal("did not reach connected state")
	}

	log.Print("Subscribing to netfield/test/#")
	if _, err := client.AddSubscription("netfield/test/#", types.QoS0); err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	log.Print("Publishing a greeting")
	payload := fmt.Sprintf("Hello from Go SDK at %s", time.Now().Format("15:04:05"))
	if err := client.PublishMessage("netfield/test/greeting", payload, types.QoS0, false); err != nil {
		log.Fatalf("publish failed: %v", err)
	}

	runTopicScan(client)

	log.Print("Press Ctrl+C to exit ...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Print("Cleaning up ...")
	if err := client.DeleteSubscription("netfield/test/#"); err != nil {
		log.Printf("unsubscribe error: %v", err)
	}
}

func runTopicScan(client *mqttclient.MQTTManager) {
	log.Print("Starting topic scan for 1 minute ...")
	if err := client.StartScan([]string{"netfield/#"}, 1); err != nil {
		log.Printf("start scan failed: %v", err)
		return
	}

	st := client.GetScanStatus()
	log.Printf("Scan status: %s, end at %s, topics: %v",
		st.Status, st.ScanEndAt, st.ExploreTopics)

	// Publish a few messages while the scan is running so we can see them.
	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Second)
		_ = client.PublishMessage(
			fmt.Sprintf("netfield/test/scan/message%d", i),
			fmt.Sprintf("scan test %d", i),
			types.QoS0, false,
		)
	}

	log.Print("Waiting for scan to complete ...")
	time.Sleep(65 * time.Second)

	st = client.GetScanStatus()
	log.Printf("Final scan status: %s", st.Status)

	topics := client.GetReceivedTopicsList()
	log.Printf("Discovered %d topics:", len(topics))
	for _, t := range topics {
		log.Printf("  - %s", t)
	}
}
