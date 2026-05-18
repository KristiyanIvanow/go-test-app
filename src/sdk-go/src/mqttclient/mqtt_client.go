// Package mqttclient implements the singleton MQTT client (MQTTManager)
// for the netfield-hub-sdk Go SDK. It mirrors the behavior of the
// sdk-node MQTTManager and sdk-python MqttClient: it owns a single
// active broker connection, transparently reconnects, supports
// subscriptions, publishing, and topic scanning.
package mqttclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	sdkerrors "github.com/KristiyanIvanow/go-test-app/src/errors"
	"github.com/KristiyanIvanow/go-test-app/src/logger"
	"github.com/KristiyanIvanow/go-test-app/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/mqttapi"
	"github.com/KristiyanIvanow/go-test-app/src/types"
)

// MQTTManager is the singleton MQTT client.
type MQTTManager struct {
	mu sync.RWMutex

	Config *models.MqttConfigModel

	// Pluggable APIs that receive matching messages.
	MyAPI                mqttapi.MqttAPI
	DesiredPropertiesAPI mqttapi.MqttAPI
	DirectMethodsAPI     mqttapi.MqttAPI

	logger logger.ILogger

	containerID string
	currentURI  string
	currentPort int

	mqttClient     mqtt.Client
	mqttScanClient mqtt.Client

	subscriptions map[string]*models.MqttSubscriptionModel
	connState     *models.MqttState

	// Scan state.
	scanStatus    types.ScanStatusEnum
	scanEndTime   time.Time
	exploreTopics []string
	receivedTopic map[string]struct{}
	scanCancel    chan struct{}
	scanRunning   bool

	// Reconnect / re-init control.
	reInitializing bool
}

var (
	instance     *MQTTManager
	instanceOnce sync.Once
)

// GetInstance returns the singleton MQTTManager instance.
func GetInstance() *MQTTManager {
	instanceOnce.Do(func() {
		instance = &MQTTManager{
			logger:        logger.Default,
			subscriptions: make(map[string]*models.MqttSubscriptionModel),
			connState:     models.NewMqttState(),
			scanStatus:    types.ScanIdle,
			receivedTopic: make(map[string]struct{}),
		}
	})
	return instance
}

// SetLogger replaces the SDK logger.
func (m *MQTTManager) SetLogger(l logger.ILogger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l != nil {
		m.logger = l
	}
}

// Init initializes the MQTT client with the supplied config and container id.
// It blocks until the first connection attempt has completed (success or
// failure to all configured brokers).
func (m *MQTTManager) Init(config *models.MqttConfigModel, containerID string) error {
	if config == nil {
		return fmt.Errorf("MQTT config is nil")
	}
	if !types.ContainerIDPattern.MatchString(containerID) {
		m.logger.Warning(fmt.Sprintf(
			"The given containerId %q violates the MQTT containerID guidelines.", containerID))
	}

	m.mu.Lock()
	m.Config = config
	m.containerID = containerID
	if len(m.Config.ServerURIs) == 0 {
		m.Config.ServerURIs = []string{"mqtt://localhost:1883"}
	}
	m.mu.Unlock()

	return m.connectToFirstAvailableBroker()
}

// DeInit gracefully disconnects the MQTT client.
func (m *MQTTManager) DeInit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mqttClient != nil && m.mqttClient.IsConnected() {
		m.mqttClient.Disconnect(250)
	}
	m.mqttClient = nil
}

// ReInit re-initializes the MQTT client with a new configuration.
func (m *MQTTManager) ReInit(config *models.MqttConfigModel, containerID string) error {
	m.mu.Lock()
	if m.reInitializing {
		m.mu.Unlock()
		m.logger.Warning("MQTT client is already re-initializing")
		return nil
	}
	m.reInitializing = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.reInitializing = false
		m.mu.Unlock()
	}()

	m.StopScan()
	m.DeInit()
	return m.Init(config, containerID)
}

// GetConnState returns the current connection state.
func (m *MQTTManager) GetConnState() *models.MqttState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.mqttClient != nil && m.mqttClient.IsConnected() {
		m.connState.ConnectionState = types.Connected
		m.connState.Message = fmt.Sprintf("Connection to %s:%d established.", m.currentURI, m.currentPort)
	} else {
		m.connState.ConnectionState = types.Disconnected
		m.connState.Message = fmt.Sprintf("Connection to %s:%d could not be established.", m.currentURI, m.currentPort)
	}
	return m.connState
}

// GetMqttClient returns the underlying paho client (may be nil).
func (m *MQTTManager) GetMqttClient() mqtt.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mqttClient
}

// IsConnected reports whether the MQTT client is currently connected.
func (m *MQTTManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mqttClient != nil && m.mqttClient.IsConnected()
}

// PublishMessage publishes a message to the broker.
func (m *MQTTManager) PublishMessage(topic, payload string, qos types.EQoS, retained bool) error {
	m.mu.RLock()
	client := m.mqttClient
	m.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return &sdkerrors.MqttClientNotConnectedError{Message: "MQTT client not connected"}
	}
	token := client.Publish(topic, byte(qos), retained, payload)
	token.Wait()
	if err := token.Error(); err != nil {
		m.logger.Error(fmt.Sprintf("Error publishing message to %s: %v", topic, err))
		return err
	}
	return nil
}

// AddSubscription subscribes to a topic at the requested QoS.
func (m *MQTTManager) AddSubscription(topic string, qos types.EQoS) (*models.MqttSubscriptionModel, error) {
	if err := validateTopicWildcard(topic); err != nil {
		return nil, err
	}

	m.mu.Lock()
	sub := models.NewMqttSubscriptionModel(topic, qos)
	m.subscriptions[topic] = sub
	client := m.mqttClient
	m.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return nil, &sdkerrors.MqttClientNotConnectedError{Message: "MQTT client not connected"}
	}

	token := client.Subscribe(topic, byte(qos), nil)
	token.Wait()
	if err := token.Error(); err != nil {
		return nil, err
	}
	return sub, nil
}

// DeleteSubscription unsubscribes from a topic.
func (m *MQTTManager) DeleteSubscription(topic string) error {
	m.mu.Lock()
	delete(m.subscriptions, topic)
	client := m.mqttClient
	m.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return &sdkerrors.MqttClientNotConnectedError{Message: "MQTT client not connected"}
	}
	token := client.Unsubscribe(topic)
	token.Wait()
	return token.Error()
}

// UpdateSubscription updates the QoS of an existing subscription.
func (m *MQTTManager) UpdateSubscription(topic string, qos types.EQoS) error {
	if err := m.DeleteSubscription(topic); err != nil {
		return err
	}
	_, err := m.AddSubscription(topic, qos)
	return err
}

// StartScan starts a topic scan for the configured duration.
func (m *MQTTManager) StartScan(exploreTopics []string, scanDurationMinutes int) error {
	m.mu.Lock()
	if m.scanRunning {
		m.mu.Unlock()
		return &sdkerrors.ScanAlreadyStartedError{Message: "Method can only be called once"}
	}
	m.mu.Unlock()

	if len(exploreTopics) == 0 {
		return &sdkerrors.ExploreTopicsError{Message: "You did not specify any explore topic"}
	}
	if scanDurationMinutes < 1 || scanDurationMinutes > 60 {
		return &sdkerrors.ScanDurationMinutesError{
			Message: "Scan duration cannot be less than 1 minute or more than 60 minutes",
		}
	}
	if !m.IsConnected() {
		return &sdkerrors.MqttClientNotConnectedError{Message: "No mqtt broker to connect to"}
	}

	m.mu.Lock()
	m.scanStatus = types.ScanRunning
	m.scanEndTime = time.Now().Add(time.Duration(scanDurationMinutes) * time.Minute)
	m.exploreTopics = exploreTopics
	m.scanCancel = make(chan struct{})
	m.scanRunning = true
	cancel := m.scanCancel
	m.mu.Unlock()

	go m.runScan(exploreTopics, scanDurationMinutes, cancel)
	return nil
}

// StopScan signals the running scan to stop.
func (m *MQTTManager) StopScan() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scanCancel != nil {
		select {
		case <-m.scanCancel:
		default:
			close(m.scanCancel)
		}
	}
	m.scanRunning = false
	m.scanStatus = types.ScanIdle
}

// GetScanStatus returns the current scan status snapshot.
func (m *MQTTManager) GetScanStatus() *models.ScanStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &models.ScanStatus{
		Status:        m.scanStatus,
		ScanEndAt:     m.scanEndTime.Format(time.RFC3339),
		ExploreTopics: m.exploreTopics,
	}
}

// GetReceivedTopicsList returns the list of topics observed during the
// last (or current) scan.
func (m *MQTTManager) GetReceivedTopicsList() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.receivedTopic))
	for t := range m.receivedTopic {
		out = append(out, t)
	}
	return out
}

// -- internals --------------------------------------------------------------

func (m *MQTTManager) connectToFirstAvailableBroker() error {
	m.mu.Lock()
	uris := append([]string(nil), m.Config.ServerURIs...)
	m.mu.Unlock()

	var lastErr error
	for _, uri := range uris {
		protocol, host, port := parseServerURI(uri)
		m.mu.Lock()
		m.currentURI = host
		m.currentPort = port
		m.connState.ConnectionState = types.Connecting
		m.mu.Unlock()

		opts, err := m.buildClientOptions(protocol, host, port, m.containerID)
		if err != nil {
			lastErr = err
			continue
		}
		opts.SetDefaultPublishHandler(m.onMessage)
		opts.OnConnect = m.onConnect
		opts.OnConnectionLost = m.onConnectionLost
		opts.OnReconnecting = func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			m.mu.Lock()
			m.connState.ConnectionState = types.Reconnecting
			m.mu.Unlock()
		}

		client := mqtt.NewClient(opts)
		token := client.Connect()
		token.Wait()
		if err := token.Error(); err != nil {
			m.logger.Error(fmt.Sprintf("Connection error with %s:%d: %v", host, port, err))
			lastErr = err
			continue
		}

		m.mu.Lock()
		m.mqttClient = client
		m.connState.ConnectionState = types.Connected
		m.connState.Message = fmt.Sprintf("Connection to %s:%d established.", host, port)
		m.mu.Unlock()

		m.logger.Information(fmt.Sprintf("### MQTT Init complete (connected to %s:%d) ###", host, port))
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no broker was available in the given server list")
	}
	return lastErr
}

func (m *MQTTManager) buildClientOptions(protocol, host string, port int, clientID string) (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", protocol, host, port))
	opts.SetClientID(clientID)
	opts.SetCleanSession(m.Config.Cleansession)
	if m.Config.KeepAliveInterval > 0 {
		opts.SetKeepAlive(time.Duration(m.Config.KeepAliveInterval) * time.Second)
	}
	if m.Config.ConnectTimeout > 0 {
		opts.SetConnectTimeout(time.Duration(m.Config.ConnectTimeout) * time.Second)
	}
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(60 * time.Second)
	opts.SetConnectRetry(true)

	if m.Config.Username != "" {
		opts.SetUsername(m.Config.Username)
		opts.SetPassword(m.Config.Password)
	}

	if w := m.Config.WillOptions; w != nil && w.TopicName != "" && w.Payload != "" {
		opts.SetWill(w.TopicName, w.Payload, byte(w.QoS), w.Retained)
	}

	if s := m.Config.SslOptions; s != nil {
		tlsCfg := &tls.Config{InsecureSkipVerify: !s.EnableServerCertAuth}
		if s.Ca != "" {
			pool := x509.NewCertPool()
			if ok := pool.AppendCertsFromPEM([]byte(s.Ca)); ok {
				tlsCfg.RootCAs = pool
			}
		}
		if s.Cert != "" && s.Key != "" {
			cert, err := tls.X509KeyPair([]byte(s.Cert), []byte(s.Key))
			if err != nil {
				return nil, fmt.Errorf("failed to load client cert/key: %w", err)
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
		opts.SetTLSConfig(tlsCfg)
	}

	return opts, nil
}

func (m *MQTTManager) onConnect(_ mqtt.Client) {
	m.mu.Lock()
	subs := make(map[string]*models.MqttSubscriptionModel, len(m.subscriptions))
	for k, v := range m.subscriptions {
		subs[k] = v
	}
	client := m.mqttClient
	m.connState.ConnectionState = types.Connected
	m.mu.Unlock()

	if client == nil {
		return
	}
	for topic, sub := range subs {
		token := client.Subscribe(topic, byte(sub.QoS), nil)
		token.Wait()
		if err := token.Error(); err != nil {
			m.logger.Warning(fmt.Sprintf("Failed to restore subscription to %s: %v", topic, err))
		}
	}
}

func (m *MQTTManager) onConnectionLost(_ mqtt.Client, err error) {
	m.mu.Lock()
	m.connState.ConnectionState = types.Disconnected
	m.mu.Unlock()
	m.logger.Warning(fmt.Sprintf("Connection lost: %v", err))
}

func (m *MQTTManager) onMessage(_ mqtt.Client, msg mqtt.Message) {
	mqttMessage := &models.MqttMessage{
		Topic:    msg.Topic(),
		Payload:  msg.Payload(),
		QoS:      types.EQoS(msg.Qos()),
		Retained: msg.Retained(),
	}
	m.logger.Debug(fmt.Sprintf("Message received with topic: %s", mqttMessage.Topic))

	m.mu.RLock()
	desired := m.DesiredPropertiesAPI
	dm := m.DirectMethodsAPI
	my := m.MyAPI
	m.mu.RUnlock()

	switch {
	case strings.HasPrefix(mqttMessage.Topic, "properties/desired/") && desired != nil:
		desired.HandleMessage(mqttMessage)
	case strings.HasPrefix(mqttMessage.Topic, "dm/") && dm != nil:
		dm.HandleMessage(mqttMessage)
	case my != nil:
		my.HandleMessage(mqttMessage)
	default:
		m.logger.Error("The MQTT API in MQTT Manager is not initialized!")
	}
}

func (m *MQTTManager) runScan(exploreTopics []string, scanDurationMinutes int, cancel <-chan struct{}) {
	defer func() {
		m.mu.Lock()
		if m.mqttScanClient != nil && m.mqttScanClient.IsConnected() {
			m.mqttScanClient.Disconnect(250)
		}
		m.mqttScanClient = nil
		m.scanStatus = types.ScanIdle
		m.scanRunning = false
		m.mu.Unlock()
	}()

	m.mu.RLock()
	uri := m.currentURI
	port := m.currentPort
	containerID := m.containerID
	m.mu.RUnlock()

	opts, err := m.buildClientOptions("tcp", uri, port, containerID+types.ScanClientIDSuffix)
	if err != nil {
		m.logger.Error(fmt.Sprintf("Scan client options error: %v", err))
		return
	}
	opts.SetDefaultPublishHandler(func(_ mqtt.Client, msg mqtt.Message) {
		m.mu.Lock()
		m.receivedTopic[msg.Topic()] = struct{}{}
		m.mu.Unlock()
		m.logger.Debug(fmt.Sprintf("### Topic received: %s", msg.Topic()))
	})

	scanClient := mqtt.NewClient(opts)
	if token := scanClient.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		m.logger.Error(fmt.Sprintf("Scan client connect error: %v", token.Error()))
		return
	}
	m.mu.Lock()
	m.mqttScanClient = scanClient
	m.mu.Unlock()

	for _, t := range exploreTopics {
		token := scanClient.Subscribe(t, 0, nil)
		token.Wait()
		if err := token.Error(); err != nil {
			m.logger.Warning(fmt.Sprintf("Scan subscribe error on %s: %v", t, err))
		}
	}

	deadline := time.After(time.Duration(scanDurationMinutes) * time.Minute)
	for {
		select {
		case <-cancel:
			return
		case <-deadline:
			return
		case <-time.After(500 * time.Millisecond):
			// keep running until cancel or deadline
		}
	}
}

// -- helpers ----------------------------------------------------------------

func parseServerURI(serverURI string) (protocol, host string, port int) {
	protocol = "tcp"
	host = ""
	port = 1883

	if strings.Contains(serverURI, "://") {
		parts := strings.SplitN(serverURI, "://", 2)
		switch parts[0] {
		case "mqtt":
			protocol = "tcp"
		case "mqtts":
			protocol = "ssl"
			port = 8883
		case "ws":
			protocol = "ws"
		case "wss":
			protocol = "wss"
		default:
			protocol = parts[0]
		}
		hostPort := parts[1]
		if strings.Contains(hostPort, ":") {
			hp := strings.SplitN(hostPort, ":", 2)
			host = hp[0]
			if p, err := strconv.Atoi(hp[1]); err == nil {
				port = p
			}
		} else {
			host = hostPort
		}
		return
	}

	if strings.Contains(serverURI, ":") {
		hp := strings.SplitN(serverURI, ":", 2)
		host = hp[0]
		if p, err := strconv.Atoi(hp[1]); err == nil {
			port = p
		}
	} else {
		host = serverURI
	}
	return
}

func validateTopicWildcard(topic string) error {
	if i := strings.Index(topic, "#"); i != -1 && i != len(topic)-1 {
		return fmt.Errorf("the character '#' is only allowed as the last character")
	}
	return nil
}

// ConvertEQoSToMqttNet maps EQoS to a paho QoS byte.
func ConvertEQoSToMqttNet(qos types.EQoS) byte {
	switch qos {
	case types.QoS1:
		return 1
	case types.QoS2:
		return 2
	default:
		return 0
	}
}

// ConvertMqttNetToEQoS maps a paho QoS byte to an EQoS.
func ConvertMqttNetToEQoS(qos byte) types.EQoS {
	switch qos {
	case 1:
		return types.QoS1
	case 2:
		return types.QoS2
	default:
		return types.QoS0
	}
}
