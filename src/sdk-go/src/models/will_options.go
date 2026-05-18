package models

import "github.com/KristiyanIvanow/go-test-app/src/types"

// WillOptions holds the MQTT "Last Will and Testament" options.
type WillOptions struct {
	TopicName string     `json:"topicName"`
	Message   string     `json:"message,omitempty"`
	Retained  bool       `json:"retained"`
	QoS       types.EQoS `json:"qos"`
	Payload   string     `json:"payload"`
}
