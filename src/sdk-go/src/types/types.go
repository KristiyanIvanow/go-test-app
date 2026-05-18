// Package types contains shared enums, constants and primitive types
// used across the netfield-hub-sdk Go SDK.
package types

import "regexp"

// EQoS represents an MQTT Quality of Service level.
type EQoS int

const (
	QoS0 EQoS = 0
	QoS1 EQoS = 1
	QoS2 EQoS = 2
)

// ConnectionState represents the state of the MQTT connection.
type ConnectionState string

const (
	Disconnected ConnectionState = "disconnected"
	Connecting   ConnectionState = "connecting"
	Connected    ConnectionState = "connected"
	Reconnecting ConnectionState = "reconnecting"
)

// ScanStatusEnum represents the state of a topic scan.
type ScanStatusEnum string

const (
	ScanIdle    ScanStatusEnum = "idle"
	ScanRunning ScanStatusEnum = "running"
)

// ContainerIDPattern is the regex used to validate MQTT container/client IDs.
var ContainerIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]{1,23}$`)

// ScanClientIDSuffix is appended to the container ID when creating the
// dedicated MQTT client used for topic scanning.
const ScanClientIDSuffix = "Sc"
