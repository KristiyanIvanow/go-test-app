// Package errors contains custom error types used by the netfield-hub-sdk.
package errors

import "fmt"

// MqttClientNotConnectedError is returned when an operation is attempted
// while the MQTT client is not connected to a broker.
type MqttClientNotConnectedError struct{ Message string }

func (e *MqttClientNotConnectedError) Error() string {
	return fmt.Sprintf("MqttClientNotConnectedError: %s", e.Message)
}

// ExploreTopicsError is returned when no explore topics were provided
// for a scan operation.
type ExploreTopicsError struct{ Message string }

func (e *ExploreTopicsError) Error() string {
	return fmt.Sprintf("ExploreTopicsError: %s", e.Message)
}

// ScanAlreadyStartedError is returned when a scan operation is started
// while another scan is already in progress.
type ScanAlreadyStartedError struct{ Message string }

func (e *ScanAlreadyStartedError) Error() string {
	return fmt.Sprintf("ScanAlreadyStartedError: %s", e.Message)
}

// ScanDurationMinutesError is returned when an invalid scan duration
// is provided (must be between 1 and 60 minutes).
type ScanDurationMinutesError struct{ Message string }

func (e *ScanDurationMinutesError) Error() string {
	return fmt.Sprintf("ScanDurationMinutesError: %s", e.Message)
}
