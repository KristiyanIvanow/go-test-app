package unit_tests_test

import (
	"errors"
	"strings"
	"testing"

	sdkerrors "github.com/KristiyanIvanow/go-test-app/src/errors"
	"github.com/KristiyanIvanow/go-test-app/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/mqttclient"
	"github.com/KristiyanIvanow/go-test-app/src/types"
)

// ---------- Singleton ---------------------------------------------------

func TestGetInstanceSingleton(t *testing.T) {
	a := mqttclient.GetInstance()
	b := mqttclient.GetInstance()
	if a != b {
		t.Fatalf("expected singleton, got two different instances")
	}
	if a == nil {
		t.Fatalf("expected non-nil singleton")
	}
}

func TestGetInstanceMultipleCalls(t *testing.T) {
	first := mqttclient.GetInstance()
	for i := 0; i < 5; i++ {
		if mqttclient.GetInstance() != first {
			t.Fatalf("call %d returned a different instance", i)
		}
	}
}

// ---------- QoS conversion ----------------------------------------------

func TestConvertEQoSRoundTrip(t *testing.T) {
	cases := []types.EQoS{types.QoS0, types.QoS1, types.QoS2}
	for _, q := range cases {
		got := mqttclient.ConvertMqttNetToEQoS(mqttclient.ConvertEQoSToMqttNet(q))
		if got != q {
			t.Errorf("round-trip mismatch: got %v want %v", got, q)
		}
	}
}

func TestConvertEQoSToMqttNet(t *testing.T) {
	cases := map[types.EQoS]byte{
		types.QoS0: 0,
		types.QoS1: 1,
		types.QoS2: 2,
	}
	for in, want := range cases {
		if got := mqttclient.ConvertEQoSToMqttNet(in); got != want {
			t.Errorf("ConvertEQoSToMqttNet(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestConvertMqttNetToEQoSDefaultsToQoS0(t *testing.T) {
	if got := mqttclient.ConvertMqttNetToEQoS(99); got != types.QoS0 {
		t.Errorf("expected unknown QoS to fall back to QoS0, got %v", got)
	}
}

// ---------- Init validation ---------------------------------------------

func TestInitNilConfigReturnsError(t *testing.T) {
	c := mqttclient.GetInstance()
	if err := c.Init(nil, "abc"); err == nil {
		t.Fatalf("expected error for nil config")
	}
}

// ---------- Operations with no broker -----------------------------------

func TestPublishMessageWithoutConnectionReturnsTypedError(t *testing.T) {
	c := mqttclient.GetInstance()
	err := c.PublishMessage("x/y", "payload", types.QoS1, false)
	if err == nil {
		t.Fatalf("expected error when not connected")
	}
	var notConn *sdkerrors.MqttClientNotConnectedError
	if !errors.As(err, &notConn) {
		t.Fatalf("expected MqttClientNotConnectedError, got %T: %v", err, err)
	}
}

func TestAddSubscriptionWithBadWildcardReturnsError(t *testing.T) {
	c := mqttclient.GetInstance()
	_, err := c.AddSubscription("a/#/b", types.QoS0)
	if err == nil {
		t.Fatalf("expected error for invalid wildcard")
	}
	if !strings.Contains(err.Error(), "#") {
		t.Errorf("error should mention '#': %v", err)
	}
}

func TestDeleteSubscriptionWithoutConnectionReturnsTypedError(t *testing.T) {
	c := mqttclient.GetInstance()
	err := c.DeleteSubscription("any/topic")
	var notConn *sdkerrors.MqttClientNotConnectedError
	if !errors.As(err, &notConn) {
		t.Fatalf("expected MqttClientNotConnectedError, got %T: %v", err, err)
	}
}

// ---------- Scan validation ---------------------------------------------

func TestStartScanRejectsEmptyTopics(t *testing.T) {
	c := mqttclient.GetInstance()
	err := c.StartScan(nil, 1)
	var typed *sdkerrors.ExploreTopicsError
	if !errors.As(err, &typed) {
		t.Fatalf("expected ExploreTopicsError, got %T: %v", err, err)
	}
}

func TestStartScanRejectsBadDuration(t *testing.T) {
	c := mqttclient.GetInstance()
	for _, d := range []int{0, -1, 61, 1000} {
		err := c.StartScan([]string{"foo/#"}, d)
		var typed *sdkerrors.ScanDurationMinutesError
		if !errors.As(err, &typed) {
			t.Errorf("duration=%d: expected ScanDurationMinutesError, got %T (%v)", d, err, err)
		}
	}
}

func TestStartScanRequiresConnection(t *testing.T) {
	c := mqttclient.GetInstance()
	err := c.StartScan([]string{"foo/#"}, 1)
	var typed *sdkerrors.MqttClientNotConnectedError
	if !errors.As(err, &typed) {
		t.Fatalf("expected MqttClientNotConnectedError, got %T (%v)", err, err)
	}
}

// ---------- Read-only state ---------------------------------------------

func TestGetScanStatusInitiallyIdle(t *testing.T) {
	c := mqttclient.GetInstance()
	st := c.GetScanStatus()
	if st == nil {
		t.Fatal("expected non-nil scan status")
	}
	if st.Status != types.ScanIdle && st.Status != types.ScanRunning {
		t.Errorf("unexpected scan status %q", st.Status)
	}
}

func TestGetReceivedTopicsListReturnsSlice(t *testing.T) {
	c := mqttclient.GetInstance()
	if c.GetReceivedTopicsList() == nil {
		t.Fatal("expected non-nil slice (possibly empty)")
	}
}

func TestGetConnStateNotNil(t *testing.T) {
	c := mqttclient.GetInstance()
	if c.GetConnState() == nil {
		t.Fatal("expected non-nil connection state")
	}
}

func TestStopScanIsIdempotent(t *testing.T) {
	c := mqttclient.GetInstance()
	c.StopScan()
	c.StopScan()
}

func TestPublicSurface(t *testing.T) {
	c := mqttclient.GetInstance()
	_ = c.IsConnected
	_ = c.PublishMessage
	_ = c.AddSubscription
	_ = c.DeleteSubscription
	_ = c.UpdateSubscription
	_ = c.StartScan
	_ = c.StopScan
	_ = c.GetScanStatus
	_ = c.GetReceivedTopicsList
	_ = c.GetConnState
	_ = c.GetMqttClient
	_ = c.SetLogger
	_ = c.Init
	_ = c.DeInit
	_ = c.ReInit

	_ = models.NewMqttConfigModel()
}
