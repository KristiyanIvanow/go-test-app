// Package directmethod provides a client for handling MQTT-based
// direct method invocations on an IoT Edge container.
package directmethod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/KristiyanIvanow/go-test-app/src/containerproperties"
	"github.com/KristiyanIvanow/go-test-app/src/logger"
	"github.com/KristiyanIvanow/go-test-app/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/mqttapi"
	"github.com/KristiyanIvanow/go-test-app/src/mqttclient"
	"github.com/KristiyanIvanow/go-test-app/src/types"
)

// DirectMethodMessage is the payload received on the dm/{moduleId}/ topic.
type DirectMethodMessage struct {
	MessageID     string `json:"messageId"`
	ContainerName string `json:"containerName"`
	MethodName    string `json:"methodName"`
	Payload       string `json:"payload"`
}

// DirectMethodResponse is published back to dm/response/{messageId}.
type DirectMethodResponse struct {
	Status  int         `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
}

// DirectMethodHandler is the callback invoked for a registered method.
type DirectMethodHandler func(payload interface{}) (DirectMethodResponse, error)

// DirectMethodsReportedProperties is the structure reported to IoT Hub.
type DirectMethodsReportedProperties struct {
	RegisteredMethods []string `json:"registeredMethods"`
}

// DirectMethodClient is a singleton client for direct method invocations.
type DirectMethodClient struct {
	moduleID                  string
	directMethodHandlers      map[string]DirectMethodHandler
	logger                    logger.ILogger
	mqttClient                *mqttclient.MQTTManager
	containerPropertiesClient *containerproperties.ContainerPropertiesClient
	subscription              *models.MqttSubscriptionModel
	cancel                    context.CancelFunc
}

var (
	dmInstance *DirectMethodClient
	dmOnce     sync.Once
)

// GetInstance returns the singleton DirectMethodClient.
func GetInstance(moduleID string, handlers map[string]DirectMethodHandler) (*DirectMethodClient, error) {
	if moduleID == "" {
		return nil, fmt.Errorf("module ID cannot be null or empty")
	}
	if handlers == nil {
		return nil, fmt.Errorf("direct method handlers cannot be nil")
	}
	dmOnce.Do(func() {
		dmInstance = &DirectMethodClient{
			moduleID:             moduleID,
			directMethodHandlers: handlers,
			logger:               logger.Default,
		}
		for name := range handlers {
			dmInstance.logger.Information(fmt.Sprintf("Registered handler for direct method: %s", name))
		}
	})
	return dmInstance, nil
}

// Init initializes the MQTT connection, container properties client and
// subscribes to the direct methods topic.
func (d *DirectMethodClient) Init(config *models.MqttConfigModel, log logger.ILogger) error {
	if config == nil {
		config = models.NewMqttConfigModel()
	}

	d.mqttClient = mqttclient.GetInstance()
	if err := d.mqttClient.Init(config, d.moduleID); err != nil {
		return err
	}

	// Wait for the MQTT connection.
	for retries := 0; retries < 10 && !d.mqttClient.IsConnected(); retries++ {
		time.Sleep(500 * time.Millisecond)
	}
	if !d.mqttClient.IsConnected() {
		return fmt.Errorf("failed to connect to MQTT broker")
	}

	cp, err := containerproperties.GetInstance(d.moduleID, log)
	if err != nil {
		return err
	}
	if err := cp.Init(config); err != nil {
		return err
	}
	d.containerPropertiesClient = cp

	if log != nil {
		d.logger = log
	}

	d.mqttClient.DirectMethodsAPI = &directMethodsAPI{
		handlers: d.directMethodHandlers,
		moduleID: d.moduleID,
		logger:   d.logger,
		mqtt:     d.mqttClient,
	}

	d.subscribeToDirectMethods()
	if err := d.reportDirectMethods(); err != nil {
		return err
	}

	d.logger.Information("DirectMethodClient initialized")
	return nil
}

func (d *DirectMethodClient) subscribeToDirectMethods() {
	topic := fmt.Sprintf("dm/%s/", d.moduleID)
	d.logger.Information(fmt.Sprintf("Starting subscription to direct methods topic: %s", topic))

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	go func() {
		retry := 1
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if !d.mqttClient.IsConnected() {
				if retry == 1 {
					d.logger.Warning(fmt.Sprintf(
						"MQTT client is not connected. Waiting before subscribing to %s", topic))
				}
				retry++
				time.Sleep(5 * time.Second)
				continue
			}
			sub, err := d.mqttClient.AddSubscription(topic, types.QoS1)
			if err != nil || sub == nil {
				retry++
				d.logger.Warning(fmt.Sprintf("Failed to subscribe to %s. Retry %d: %v", topic, retry, err))
				time.Sleep(5 * time.Second)
				continue
			}
			d.subscription = sub
			d.logger.Information(fmt.Sprintf("Successfully subscribed to %s", topic))
			return
		}
	}()
}

func (d *DirectMethodClient) reportDirectMethods() error {
	if d.containerPropertiesClient == nil {
		return fmt.Errorf("ContainerPropertiesClient is not initialized")
	}
	names := make([]string, 0, len(d.directMethodHandlers))
	for n := range d.directMethodHandlers {
		names = append(names, n)
	}
	reported := map[string]interface{}{"registeredMethods": names}
	if err := d.containerPropertiesClient.UpdateReportedProperties(reported); err != nil {
		d.logger.Error(fmt.Sprintf("Failed to report direct methods to IoT Hub: %v", err))
		return err
	}
	d.logger.Information(fmt.Sprintf(
		"Reported %d direct methods to IoT Hub: %s", len(names), strings.Join(names, ", ")))
	return nil
}

// Stop cancels the background subscription retry loop.
func (d *DirectMethodClient) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
}

// directMethodsAPI implements mqttapi.MqttAPI for incoming direct methods.
type directMethodsAPI struct {
	handlers map[string]DirectMethodHandler
	moduleID string
	logger   logger.ILogger
	mqtt     *mqttclient.MQTTManager
}

func (a *directMethodsAPI) HandleMessage(message *models.MqttMessage) {
	go func() {
		if err := a.process(message); err != nil {
			a.logger.Error(fmt.Sprintf("Error processing direct method message: %v", err))
		}
	}()
}

func (a *directMethodsAPI) process(message *models.MqttMessage) error {
	parts := strings.Split(message.Topic, "/")
	if len(parts) < 2 {
		a.logger.Warning(fmt.Sprintf("Invalid direct method topic format: %s", message.Topic))
		return nil
	}
	if parts[1] != a.moduleID {
		a.logger.Warning(fmt.Sprintf("Direct method invocation for different module: %s", parts[1]))
		return nil
	}

	var dm DirectMethodMessage
	if err := json.Unmarshal(message.Payload, &dm); err != nil {
		a.logger.Warning("Failed to deserialize direct method message payload")
		return nil
	}
	if strings.TrimSpace(dm.MessageID) == "" {
		a.logger.Warning("Direct method message is missing MessageId")
		return nil
	}
	if strings.TrimSpace(dm.MethodName) == "" {
		a.logger.Warning("Direct method message is null or missing method name")
		return nil
	}

	handler, ok := a.handlers[dm.MethodName]
	if !ok {
		a.logger.Warning(fmt.Sprintf("No handler registered for direct method: %s", dm.MethodName))
		a.publishResponse(dm.MessageID, DirectMethodResponse{
			Status:  404,
			Payload: map[string]string{"message": fmt.Sprintf("Method with name %q not found", dm.MethodName)},
		})
		return nil
	}

	var methodPayload interface{}
	if strings.TrimSpace(dm.Payload) != "" {
		if err := json.Unmarshal([]byte(dm.Payload), &methodPayload); err != nil {
			a.logger.Warning(fmt.Sprintf("Failed to parse direct method payload as JSON: %v", err))
			a.publishResponse(dm.MessageID, DirectMethodResponse{
				Status:  400,
				Payload: map[string]string{"message": "Invalid JSON format"},
			})
			return nil
		}
		if _, isObject := methodPayload.(map[string]interface{}); !isObject {
			a.logger.Warning("Direct method payload is not a JSON object")
			a.publishResponse(dm.MessageID, DirectMethodResponse{
				Status:  400,
				Payload: map[string]string{"message": "Payload must be a JSON object"},
			})
			return nil
		}
	}

	response, err := handler(methodPayload)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Direct method %s handler error: %v", dm.MethodName, err))
		response = DirectMethodResponse{
			Status:  500,
			Payload: map[string]string{"message": err.Error()},
		}
	}
	a.publishResponse(dm.MessageID, response)
	return nil
}

func (a *directMethodsAPI) publishResponse(messageID string, response DirectMethodResponse) {
	topic := fmt.Sprintf("dm/response/%s", messageID)
	body, err := json.Marshal(response)
	if err != nil {
		a.logger.Error(fmt.Sprintf("Failed to serialize direct method response: %v", err))
		return
	}
	if err := a.mqtt.PublishMessage(topic, string(body), types.QoS1, false); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to publish direct method response for %s: %v", messageID, err))
	}
}

var _ mqttapi.MqttAPI = (*directMethodsAPI)(nil)
