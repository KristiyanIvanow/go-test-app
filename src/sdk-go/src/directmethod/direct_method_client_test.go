package directmethod

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/models"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/mqttclient"
)

type captureLogger struct {
	infos    []string
	warnings []string
	errors   []string
	debugs   []string
}

func (c *captureLogger) Information(message string) { c.infos = append(c.infos, message) }
func (c *captureLogger) Warning(message string)     { c.warnings = append(c.warnings, message) }
func (c *captureLogger) Error(message string)       { c.errors = append(c.errors, message) }
func (c *captureLogger) Debug(message string)       { c.debugs = append(c.debugs, message) }

func resetDirectMethodSingleton() {
	dmInstance = nil
	dmOnce = sync.Once{}
}

func hasLogContaining(logs []string, token string) bool {
	for _, line := range logs {
		if strings.Contains(line, token) {
			return true
		}
	}
	return false
}

func TestProcessInvalidTopicFormatLogsWarning(t *testing.T) {
	log := &captureLogger{}
	api := &directMethodsAPI{
		handlers: map[string]DirectMethodHandler{},
		moduleID: "mod1",
		logger:   log,
		mqtt:     mqttclient.GetInstance(),
	}

	msg := &models.MqttMessage{Topic: "dm", Payload: []byte("{}")}
	if err := api.process(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLogContaining(log.warnings, "Invalid direct method topic format") {
		t.Fatalf("expected invalid topic warning, got: %#v", log.warnings)
	}
}

func TestProcessDifferentModuleLogsWarning(t *testing.T) {
	log := &captureLogger{}
	api := &directMethodsAPI{
		handlers: map[string]DirectMethodHandler{},
		moduleID: "expected",
		logger:   log,
		mqtt:     mqttclient.GetInstance(),
	}

	msg := &models.MqttMessage{Topic: "dm/other/", Payload: []byte("{}")}
	if err := api.process(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLogContaining(log.warnings, "Direct method invocation for different module") {
		t.Fatalf("expected different module warning, got: %#v", log.warnings)
	}
}

func TestProcessMissingFieldsLogsWarnings(t *testing.T) {
	t.Run("missing messageId", func(t *testing.T) {
		log := &captureLogger{}
		api := &directMethodsAPI{
			handlers: map[string]DirectMethodHandler{},
			moduleID: "mod1",
			logger:   log,
			mqtt:     mqttclient.GetInstance(),
		}
		msg := &models.MqttMessage{Topic: "dm/mod1/", Payload: []byte(`{"methodName":"Ping","payload":"{}"}`)}
		_ = api.process(msg)
		if !hasLogContaining(log.warnings, "missing MessageId") {
			t.Fatalf("expected missing MessageId warning, got: %#v", log.warnings)
		}
	})

	t.Run("missing methodName", func(t *testing.T) {
		log := &captureLogger{}
		api := &directMethodsAPI{
			handlers: map[string]DirectMethodHandler{},
			moduleID: "mod1",
			logger:   log,
			mqtt:     mqttclient.GetInstance(),
		}
		msg := &models.MqttMessage{Topic: "dm/mod1/", Payload: []byte(`{"messageId":"m1","payload":"{}"}`)}
		_ = api.process(msg)
		if !hasLogContaining(log.warnings, "missing method name") {
			t.Fatalf("expected missing method name warning, got: %#v", log.warnings)
		}
	})
}

func TestProcessInvalidPayloadFormsLogWarnings(t *testing.T) {
	t.Run("payload not valid json", func(t *testing.T) {
		log := &captureLogger{}
		api := &directMethodsAPI{
			handlers: map[string]DirectMethodHandler{
				"Ping": func(payload interface{}) (DirectMethodResponse, error) {
					return DirectMethodResponse{Status: 200}, nil
				},
			},
			moduleID: "mod1",
			logger:   log,
			mqtt:     mqttclient.GetInstance(),
		}
		msg := &models.MqttMessage{Topic: "dm/mod1/", Payload: []byte(`{"messageId":"m1","methodName":"Ping","payload":"not-json"}`)}
		_ = api.process(msg)
		if !hasLogContaining(log.warnings, "Failed to parse direct method payload as JSON") {
			t.Fatalf("expected invalid payload warning, got: %#v", log.warnings)
		}
	})

	t.Run("payload json is not object", func(t *testing.T) {
		log := &captureLogger{}
		api := &directMethodsAPI{
			handlers: map[string]DirectMethodHandler{
				"Ping": func(payload interface{}) (DirectMethodResponse, error) {
					return DirectMethodResponse{Status: 200}, nil
				},
			},
			moduleID: "mod1",
			logger:   log,
			mqtt:     mqttclient.GetInstance(),
		}
		msg := &models.MqttMessage{Topic: "dm/mod1/", Payload: []byte(`{"messageId":"m1","methodName":"Ping","payload":"[1,2,3]"}`)}
		_ = api.process(msg)
		if !hasLogContaining(log.warnings, "payload is not a JSON object") {
			t.Fatalf("expected non-object payload warning, got: %#v", log.warnings)
		}
	})
}

func TestProcessInvokesHandlerWithParsedPayload(t *testing.T) {
	log := &captureLogger{}
	called := false
	api := &directMethodsAPI{
		handlers: map[string]DirectMethodHandler{
			"Apply": func(payload interface{}) (DirectMethodResponse, error) {
				called = true
				payloadMap, ok := payload.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map payload, got %T", payload)
				}
				if payloadMap["x"] != float64(1) {
					t.Fatalf("expected payload x=1, got %#v", payloadMap["x"])
				}
				return DirectMethodResponse{Status: 200, Payload: map[string]string{"ok": "true"}}, nil
			},
		},
		moduleID: "mod1",
		logger:   log,
		mqtt:     mqttclient.GetInstance(),
	}

	msg := &models.MqttMessage{Topic: "dm/mod1/", Payload: []byte(`{"messageId":"m1","methodName":"Apply","payload":"{\"x\":1}"}`)}
	if err := api.process(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestProcessHandlerErrorLogsError(t *testing.T) {
	log := &captureLogger{}
	api := &directMethodsAPI{
		handlers: map[string]DirectMethodHandler{
			"Fail": func(payload interface{}) (DirectMethodResponse, error) {
				return DirectMethodResponse{}, errors.New("boom")
			},
		},
		moduleID: "mod1",
		logger:   log,
		mqtt:     mqttclient.GetInstance(),
	}

	msg := &models.MqttMessage{Topic: "dm/mod1/", Payload: []byte(`{"messageId":"m1","methodName":"Fail","payload":"{}"}`)}
	if err := api.process(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLogContaining(log.errors, "handler error") {
		t.Fatalf("expected handler error log, got: %#v", log.errors)
	}
}

func TestReportDirectMethodsRequiresContainerPropertiesClient(t *testing.T) {
	client := &DirectMethodClient{
		directMethodHandlers: map[string]DirectMethodHandler{"Ping": nil},
		logger:               &captureLogger{},
	}

	err := client.reportDirectMethods()
	if err == nil {
		t.Fatal("expected error when containerPropertiesClient is nil")
	}
	if !strings.Contains(err.Error(), "ContainerPropertiesClient is not initialized") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetInstanceInitializesSingleton(t *testing.T) {
	resetDirectMethodSingleton()
	handlers := map[string]DirectMethodHandler{
		"Ping": func(payload interface{}) (DirectMethodResponse, error) {
			return DirectMethodResponse{Status: 200}, nil
		},
	}

	a, err := GetInstance("mod1", handlers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := GetInstance("mod1", handlers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != b {
		t.Fatal("expected singleton instances to match")
	}
}
