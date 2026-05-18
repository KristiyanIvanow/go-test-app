package models

import "github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/types"

// MqttState represents the current MQTT connection state.
type MqttState struct {
	ConnectionState types.ConnectionState `json:"connectionState"`
	Message         string                `json:"message,omitempty"`
}

// NewMqttState creates a new MqttState defaulting to Disconnected.
func NewMqttState() *MqttState {
	return &MqttState{ConnectionState: types.Disconnected}
}
