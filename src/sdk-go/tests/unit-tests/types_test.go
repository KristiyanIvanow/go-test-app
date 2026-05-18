package unit_tests_test

import (
	"testing"

	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/types"
)

func TestEQoSValues(t *testing.T) {
	if types.QoS0 != 0 || types.QoS1 != 1 || types.QoS2 != 2 {
		t.Errorf("EQoS values changed: %d %d %d", types.QoS0, types.QoS1, types.QoS2)
	}
}

func TestConnectionStateConstants(t *testing.T) {
	cases := map[types.ConnectionState]string{
		types.Disconnected: "disconnected",
		types.Connecting:   "connecting",
		types.Connected:    "connected",
		types.Reconnecting: "reconnecting",
	}
	for state, want := range cases {
		if string(state) != want {
			t.Errorf("ConnectionState %q != %q", state, want)
		}
	}
}

func TestScanStatusEnumConstants(t *testing.T) {
	if types.ScanIdle != "idle" {
		t.Errorf("ScanIdle = %q, want %q", types.ScanIdle, "idle")
	}
	if types.ScanRunning != "running" {
		t.Errorf("ScanRunning = %q, want %q", types.ScanRunning, "running")
	}
}

func TestContainerIDPattern(t *testing.T) {
	valid := []string{"a", "abc", "ABC123", "Container01"}
	invalid := []string{"", "with space", "with-dash", "with.dot", "way_too_long_for_an_id_xx"}

	for _, v := range valid {
		if !types.ContainerIDPattern.MatchString(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}
	for _, v := range invalid {
		if types.ContainerIDPattern.MatchString(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}

func TestScanClientIDSuffix(t *testing.T) {
	if types.ScanClientIDSuffix != "Sc" {
		t.Errorf("ScanClientIDSuffix changed: %q", types.ScanClientIDSuffix)
	}
}
