package models

// MqttConfigModel holds the configuration used to initialize the MQTT client.
type MqttConfigModel struct {
	SchemaVersion     int          `json:"schemaVersion"`
	UseHostConfig     bool         `json:"useHostConfig"`
	KeepAliveInterval int          `json:"keepAliveInterval,omitempty"`
	Cleansession      bool         `json:"cleansession"`
	WillOptions       *WillOptions `json:"willOptions,omitempty"`
	Username          string       `json:"username,omitempty"`
	Password          string       `json:"password,omitempty"`
	BinaryPwd         string       `json:"binarypwd,omitempty"`
	ConnectTimeout    int          `json:"connectTimeout,omitempty"`
	SslOptions        *SslOptions  `json:"sslOptions,omitempty"`
	ServerURIs        []string     `json:"serverURIs"`
	MqttVersion       int          `json:"mqttVersion"`
}

// NewMqttConfigModel creates a default MqttConfigModel.
func NewMqttConfigModel() *MqttConfigModel {
	return &MqttConfigModel{
		SchemaVersion:     1,
		UseHostConfig:     true,
		KeepAliveInterval: 60,
		Cleansession:      false,
		ConnectTimeout:    30,
		ServerURIs:        []string{"mqtt://localhost:1883", "mqtts://mosquitto:1883"},
		MqttVersion:       3,
	}
}
