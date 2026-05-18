// Package mqttapi defines the MqttAPI interface used by MqttClient
// to dispatch incoming MQTT messages to higher level handlers
// (custom user handler, desired-properties handler, direct-method handler).
package mqttapi

import "github.com/KristiyanIvanow/go-test-app/src/models"

// MqttAPI is the interface implemented by message handlers
// registered with the MQTT client.
type MqttAPI interface {
	HandleMessage(message *models.MqttMessage)
}
