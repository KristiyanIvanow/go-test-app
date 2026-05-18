//go:build integration

// Integration test for MqttClient. Requires a running MQTT broker.
//
// Run with:
//
//   go test -tags=integration ./tests/integration-tests/mqtt_client/...
//
// Override the broker URI via env: MQTT_URI=mqtt://mosquitto:1883
//
// The test is automatically skipped when no broker is reachable so that
// CI without a sidecar mosquitto does not fail spuriously.
package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/KristiyanIvanow/go-test-app/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/mqttclient"
	"github.com/KristiyanIvanow/go-test-app/src/types"
)

func brokerURI() string {
	if v := os.Getenv("MQTT_URI"); v != "" {
		return v
	}
	return "mqtt://127.0.0.1:1883"
}

// skipIfBrokerUnreachable does a TCP dial and skips the test when the
// broker port is not accepting connections.
func skipIfBrokerUnreachable(t *testing.T) {
	t.Helper()
	uri := brokerURI()
	u, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("invalid broker URI %q: %v", uri, err)
	}
	host := u.Host
	if host == "" {
		host = uri
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		t.Skipf("broker %s not reachable: %v", host, err)
	}
	_ = conn.Close()
}

type collectAPI struct {
	mu       sync.Mutex
	messages []*models.MqttMessage
}

func (c *collectAPI) HandleMessage(m *models.MqttMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, m)
}

func (c *collectAPI) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.messages)
}

func TestIntegrationConnectPublishSubscribe(t *testing.T) {
	skipIfBrokerUnreachable(t)

	cfg := models.NewMqttConfigModel()
	cfg.ServerURIs = []string{brokerURI()}

	client := mqttclient.GetInstance()
	api := &collectAPI{}
	client.MyAPI = api

	if err := client.Init(cfg, "GoItTest1"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	t.Cleanup(client.DeInit)

	if !client.IsConnected() {
		t.Fatal("expected client to be connected after Init")
	}

	topic := fmt.Sprintf("netfield/it/go/%d", time.Now().UnixNano())
	if _, err := client.AddSubscription(topic, types.QoS1); err != nil {
		t.Fatalf("AddSubscription failed: %v", err)
	}
	t.Cleanup(func() { _ = client.DeleteSubscription(topic) })

	payload := "hello from integration test"
	if err := client.PublishMessage(topic, payload, types.QoS1, false); err != nil {
		t.Fatalf("PublishMessage failed: %v", err)
	}

	// Wait up to 5s for the message to round-trip back to us.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if api.count() > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if api.count() == 0 {
		t.Fatalf("did not receive published message within timeout")
	}
}

func TestIntegrationScanCollectsTopics(t *testing.T) {
	skipIfBrokerUnreachable(t)

	cfg := models.NewMqttConfigModel()
	cfg.ServerURIs = []string{brokerURI()}

	client := mqttclient.GetInstance()
	if err := client.Init(cfg, "GoItScan1"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	t.Cleanup(client.DeInit)

	// Smallest allowed scan window is 1 minute, so we don't wait for the
	// scan to complete; we only verify it starts and we can stop it.
	if err := client.StartScan([]string{"netfield/it/scan/#"}, 1); err != nil {
		t.Fatalf("StartScan failed: %v", err)
	}

	st := client.GetScanStatus()
	if st.Status != types.ScanRunning {
		t.Errorf("expected scan to be running, got %q", st.Status)
	}

	// Publish a few messages on the scanned wildcard.
	for i := 0; i < 3; i++ {
		_ = client.PublishMessage(
			fmt.Sprintf("netfield/it/scan/m%d", i),
			"x", types.QoS0, false,
		)
	}

	time.Sleep(2 * time.Second)
	client.StopScan()

	st = client.GetScanStatus()
	if st.Status != types.ScanIdle {
		t.Errorf("expected scan to be idle after StopScan, got %q", st.Status)
	}
}
