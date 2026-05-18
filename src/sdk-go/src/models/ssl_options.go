package models

// SslOptions holds the TLS/SSL options used by the MQTT client.
type SslOptions struct {
	Key                  string `json:"key,omitempty"`
	Cert                 string `json:"cert,omitempty"`
	Ca                   string `json:"ca,omitempty"`
	EnableServerCertAuth bool   `json:"enableServerCertAuth"`
}
