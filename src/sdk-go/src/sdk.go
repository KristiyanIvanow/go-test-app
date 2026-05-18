// Package sdk is the root package of the netfield-hub-sdk Go SDK.
//
// The SDK is organized into subpackages under src/:
//
//   - types               : enums, constants and shared types
//   - models              : data models (config, message, state, ...)
//   - errors              : custom SDK error types
//   - logger              : logger interface and default implementation
//   - mqttapi             : MqttAPI base interface used to dispatch messages
//   - mqttclient          : MQTT broker client (singleton MQTTManager)
//   - containerproperties : Container properties client (reported / desired)
//   - directmethod        : Direct method invocation client
//
package sdk
