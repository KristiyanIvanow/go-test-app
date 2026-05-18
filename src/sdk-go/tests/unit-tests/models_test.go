package unit_tests_test

import (
	"encoding/json"
	"testing"

	"github.com/KristiyanIvanow/go-test-app/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/types"
)

func TestNewMqttConfigModelDefaults(t *testing.T) {
	c := models.NewMqttConfigModel()
	if c == nil {
		t.Fatal("expected non-nil config")
	}
	if c.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", c.SchemaVersion)
	}
	if !c.UseHostConfig {
		t.Errorf("UseHostConfig should default to true")
	}
	if c.KeepAliveInterval != 60 {
		t.Errorf("KeepAliveInterval = %d, want 60", c.KeepAliveInterval)
	}
	if c.ConnectTimeout != 30 {
		t.Errorf("ConnectTimeout = %d, want 30", c.ConnectTimeout)
	}
	if c.MqttVersion != 3 {
		t.Errorf("MqttVersion = %d, want 3", c.MqttVersion)
	}
	if len(c.ServerURIs) == 0 {
		t.Errorf("expected at least one default ServerURI, got empty slice")
	}
	if c.ServerURIs[0] != "mqtt://localhost:1883" {
		t.Errorf("first default ServerURI = %q, want mqtt://localhost:1883", c.ServerURIs[0])
	}
}

func TestMqttConfigModelAcceptsCustomValues(t *testing.T) {
	c := &models.MqttConfigModel{
		ServerURIs:  []string{"mqtt://broker:1883", "mqtts://secure:8883"},
		Username:    "u",
		Password:    "p",
		MqttVersion: 5,
		SslOptions: &models.SslOptions{
			Cert:                 "C",
			Key:                  "K",
			Ca:                   "CA",
			EnableServerCertAuth: true,
		},
		WillOptions: &models.WillOptions{
			TopicName: "device/status",
			Payload:   "offline",
			QoS:       types.QoS1,
			Retained:  true,
		},
	}

	if len(c.ServerURIs) != 2 {
		t.Errorf("ServerURIs len = %d, want 2", len(c.ServerURIs))
	}
	if c.SslOptions == nil || !c.SslOptions.EnableServerCertAuth {
		t.Errorf("SslOptions not set as expected")
	}
	if c.WillOptions == nil || c.WillOptions.QoS != types.QoS1 {
		t.Errorf("WillOptions not set as expected")
	}
}

func TestMqttSubscriptionModelConstructor(t *testing.T) {
	s := models.NewMqttSubscriptionModel("foo/bar", types.QoS2)
	if s.Topic != "foo/bar" || s.QoS != types.QoS2 {
		t.Errorf("unexpected subscription: %+v", s)
	}
}

func TestMqttStateDefaults(t *testing.T) {
	s := models.NewMqttState()
	if s.ConnectionState != types.Disconnected {
		t.Errorf("default state = %q, want %q", s.ConnectionState, types.Disconnected)
	}
}

func TestMqttMessageJSONRoundTrip(t *testing.T) {
	m := &models.MqttMessage{
		Topic:    "t",
		Payload:  []byte("hello"),
		QoS:      types.QoS1,
		Retained: true,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out models.MqttMessage
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if out.Topic != m.Topic || out.QoS != m.QoS || out.Retained != m.Retained {
		t.Errorf("round trip mismatch: %+v", out)
	}
}

func TestScanStatusFields(t *testing.T) {
	s := &models.ScanStatus{
		Status:        types.ScanRunning,
		ScanEndAt:     "2026-01-01T00:00:00Z",
		ExploreTopics: []string{"a/#", "b/+"},
	}
	if s.Status != types.ScanRunning {
		t.Errorf("Status = %q", s.Status)
	}
	if len(s.ExploreTopics) != 2 {
		t.Errorf("ExploreTopics len = %d", len(s.ExploreTopics))
	}
}
