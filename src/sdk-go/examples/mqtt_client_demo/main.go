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

// mqttClientDemoAPI prints every message the broker pushes to subscribed topics.
type mqttClientDemoAPI struct{}

func (mqttClientDemoAPI) HandleMessage(message *models.MqttMessage) {
	fmt.Println("=== Message Received ===")
	fmt.Printf("Topic:    %s\n", message.Topic)
	fmt.Printf("Payload:  %s\n", string(message.Payload))
	fmt.Printf("QoS:      %d\n", message.QoS)
	fmt.Printf("Retained: %v\n", message.Retained)
	fmt.Println("========================")
}

func main() {
	uri := os.Getenv("MQTT_URI")
	if uri == "" {
		uri = "mqtt://127.0.0.1:1883"
	}

	config := models.NewMqttConfigModel()
	config.ServerURIs = []string{uri}
	config.MqttVersion = 4
	config.ConnectTimeout = 30
	config.KeepAliveInterval = 60

	client := mqttclient.GetInstance()
	client.MyAPI = mqttClientDemoAPI{}

	log.Printf("Connecting to %s ...", uri)
	if err := client.Init(config, "TestApp123"); err != nil {
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
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

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

	status := client.GetScanStatus()
	log.Printf("Scan status: %s, end at %s, topics: %v",
		status.Status, status.ScanEndAt, status.ExploreTopics)

	for index := 0; index < 3; index++ {
		time.Sleep(2 * time.Second)
		_ = client.PublishMessage(
			fmt.Sprintf("netfield/test/scan/message%d", index),
			fmt.Sprintf("scan test %d", index),
			types.QoS0, false,
		)
	}

	log.Print("Waiting for scan to complete ...")
	time.Sleep(65 * time.Second)

	status = client.GetScanStatus()
	log.Printf("Final scan status: %s", status.Status)

	topics := client.GetReceivedTopicsList()
	log.Printf("Discovered %d topics:", len(topics))
	for _, topic := range topics {
		log.Printf("  - %s", topic)
	}
}
