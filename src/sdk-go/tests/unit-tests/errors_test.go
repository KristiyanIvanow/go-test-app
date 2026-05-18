package unit_tests_test

import (
	"errors"
	"strings"
	"testing"

	sdkerrors "github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/errors"
)

func TestErrorMessagesIncludeContext(t *testing.T) {
	cases := map[error]string{
		&sdkerrors.MqttClientNotConnectedError{Message: "not connected"}: "MqttClientNotConnectedError",
		&sdkerrors.ExploreTopicsError{Message: "no topics"}:              "ExploreTopicsError",
		&sdkerrors.ScanAlreadyStartedError{Message: "running"}:           "ScanAlreadyStartedError",
		&sdkerrors.ScanDurationMinutesError{Message: "out of range"}:     "ScanDurationMinutesError",
	}
	for err, prefix := range cases {
		if !strings.Contains(err.Error(), prefix) {
			t.Errorf("%T.Error() = %q, want it to contain %q", err, err.Error(), prefix)
		}
	}
}

func TestErrorsAsTypeAssertion(t *testing.T) {
	original := &sdkerrors.MqttClientNotConnectedError{Message: "x"}
	var err error = original

	var target *sdkerrors.MqttClientNotConnectedError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should succeed for matching custom error type")
	}
	if target.Message != "x" {
		t.Errorf("Message round-trip failed: got %q", target.Message)
	}
}

func TestErrorsAsRejectsDifferentType(t *testing.T) {
	err := &sdkerrors.ExploreTopicsError{Message: "y"}
	var target *sdkerrors.ScanDurationMinutesError
	if errors.As(err, &target) {
		t.Errorf("errors.As should not match unrelated error types")
	}
}
