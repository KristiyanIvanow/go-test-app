// Package containerproperties provides a client for managing IoT Edge
// container "reported" and "desired" properties via MQTT.
package containerproperties

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/logger"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/mqttapi"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/mqttclient"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/types"
)

// DesiredPropertiesHandler is invoked when desired properties are received.
type DesiredPropertiesHandler func(payload map[string]interface{})

// ContainerPropertiesClient is a singleton client for container properties.
type ContainerPropertiesClient struct {
	moduleID                string
	desiredPropertiesHandler DesiredPropertiesHandler
	logger                   logger.ILogger
	mqttClient               *mqttclient.MQTTManager
	subscription             *models.MqttSubscriptionModel
	cancel                   context.CancelFunc
}

var (
	cpInstance *ContainerPropertiesClient
	cpOnce     sync.Once
)

// GetInstance returns the singleton ContainerPropertiesClient instance.
func GetInstance(moduleID string, log logger.ILogger) (*ContainerPropertiesClient, error) {
	if moduleID == "" {
		return nil, fmt.Errorf("module ID cannot be null or empty")
	}
	cpOnce.Do(func() {
		l := log
		if l == nil {
			l = logger.Default
		}
		cpInstance = &ContainerPropertiesClient{
			moduleID:   moduleID,
			logger:     l,
			mqttClient: mqttclient.GetInstance(),
		}
	})
	return cpInstance, nil
}

// Init initializes the underlying MQTT client.
func (c *ContainerPropertiesClient) Init(config *models.MqttConfigModel) error {
	if config == nil {
		config = models.NewMqttConfigModel()
	}
	return c.mqttClient.Init(config, c.moduleID)
}

// UpdateReportedProperties publishes the reported properties for this module.
func (c *ContainerPropertiesClient) UpdateReportedProperties(reported map[string]interface{}) error {
	if reported == nil {
		return fmt.Errorf("reported properties must be a valid object")
	}
	payload, err := json.Marshal(reported)
	if err != nil {
		return fmt.Errorf("failed to serialize reported properties: %w", err)
	}
	topic := fmt.Sprintf("properties/reported/%s", c.moduleID)
	if err := c.mqttClient.PublishMessage(topic, string(payload), types.QoS1, false); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to update reported properties: %v", err))
		return err
	}
	c.logger.Debug(fmt.Sprintf("Published reported properties to %s", topic))
	return nil
}

// SetDesiredPropertyUpdateCallback registers a handler for desired properties.
func (c *ContainerPropertiesClient) SetDesiredPropertyUpdateCallback(handler DesiredPropertiesHandler) error {
	if handler == nil {
		return fmt.Errorf("desired properties handler cannot be nil")
	}
	c.desiredPropertiesHandler = handler
	c.mqttClient.DesiredPropertiesAPI = &desiredPropertiesAPI{handler: handler, log: c.logger}
	c.logger.Information("Desired properties handler registered")
	c.subscribeToDesiredProperties()
	return nil
}

func (c *ContainerPropertiesClient) subscribeToDesiredProperties() {
	topic := fmt.Sprintf("properties/desired/%s/", c.moduleID)
	c.logger.Information(fmt.Sprintf("Starting subscription to desired properties topic: %s", topic))

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	go func() {
		retry := 1
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if !c.mqttClient.IsConnected() {
				if retry == 1 {
					c.logger.Warning(fmt.Sprintf(
						"MQTT client is not connected. Waiting before subscribing to %s", topic))
				}
				retry++
				time.Sleep(5 * time.Second)
				continue
			}
			sub, err := c.mqttClient.AddSubscription(topic, types.QoS1)
			if err != nil || sub == nil {
				retry++
				c.logger.Warning(fmt.Sprintf("Failed to subscribe to %s. Retry %d: %v", topic, retry, err))
				time.Sleep(5 * time.Second)
				continue
			}
			c.subscription = sub
			c.logger.Information(fmt.Sprintf("Successfully subscribed to %s", topic))
			return
		}
	}()
}

// Stop cancels the background subscription retry loop.
func (c *ContainerPropertiesClient) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// desiredPropertiesAPI implements mqttapi.MqttAPI for desired properties.
type desiredPropertiesAPI struct {
	handler DesiredPropertiesHandler
	log     logger.ILogger
}

func (d *desiredPropertiesAPI) HandleMessage(message *models.MqttMessage) {
	var payload map[string]interface{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		d.log.Error(fmt.Sprintf("Error processing desired properties message: %v", err))
		return
	}
	if d.handler != nil {
		d.handler(payload)
	}
}

// Compile-time interface assertion.
var _ mqttapi.MqttAPI = (*desiredPropertiesAPI)(nil)
