package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/directmethod"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/logger"
	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/models"
)

// Example: DirectMethodClient initialization and usage in Go SDK
// This demonstrates how to:
// 1. Define direct method handlers
// 2. Initialize DirectMethodClient
// 3. Handle incoming direct method requests
// 4. Return responses

func main() {
	// Step 1: Create a logger (optional)
	appLogger := logger.NewConsoleLogger()

	// Step 2: Define your direct method handlers
	// Handlers are functions that process incoming direct method requests
	handlers := map[string]directmethod.DirectMethodHandler{
		// Simple status query - no payload required
		"GetStatus": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			appLogger.Information("GetStatus called")
			return directmethod.DirectMethodResponse{
				Status: 200,
				Payload: map[string]interface{}{
					"status":        "running",
					"uptime":        "2h 30m",
					"last_reported": time.Now().Format(time.RFC3339),
				},
			}, nil
		},

		// Configure method - expects JSON payload with configuration
		"Configure": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			appLogger.Information(fmt.Sprintf("Configure called with payload: %v", payload))

			// Parse payload as map[string]interface{}
			config, ok := payload.(map[string]interface{})
			if !ok {
				return directmethod.DirectMethodResponse{
					Status: 400,
					Payload: map[string]string{
						"error": "Invalid payload format",
					},
				}, nil
			}

			// Extract configuration parameters
			interval, exists := config["interval"]
			if !exists {
				return directmethod.DirectMethodResponse{
					Status: 400,
					Payload: map[string]string{
						"error": "Missing 'interval' parameter",
					},
				}, nil
			}

			appLogger.Information(fmt.Sprintf("Configuration updated: interval = %v", interval))

			return directmethod.DirectMethodResponse{
				Status: 200,
				Payload: map[string]interface{}{
					"message":  "Configuration applied successfully",
					"interval": interval,
				},
			}, nil
		},

		// Reboot method - requires confirmation parameter
		"Reboot": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			appLogger.Information(fmt.Sprintf("Reboot called with payload: %v", payload))

			config, ok := payload.(map[string]interface{})
			if !ok {
				return directmethod.DirectMethodResponse{
					Status:  400,
					Payload: map[string]string{"error": "Invalid payload"},
				}, nil
			}

			confirm, exists := config["confirmed"]
			if !exists || confirm != true {
				return directmethod.DirectMethodResponse{
					Status: 400,
					Payload: map[string]string{
						"error": "Reboot confirmation required (set confirmed: true)",
					},
				}, nil
			}

			appLogger.Information("Rebooting device...")
			// In a real application, you would trigger the actual reboot here
			// For this example, we just return success
			return directmethod.DirectMethodResponse{
				Status: 200,
				Payload: map[string]interface{}{
					"message":        "Device rebooting",
					"reboot_at":      time.Now().Add(5 * time.Second).Format(time.RFC3339),
					"estimated_time": "30 seconds",
				},
			}, nil
		},

		// Telemetry method - retrieves sensor data
		"GetTelemetry": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			appLogger.Information("GetTelemetry called")

			telemetry := map[string]interface{}{
				"temperature": 23.5,
				"humidity":    45.2,
				"pressure":    1013.25,
				"timestamp":   time.Now().Format(time.RFC3339),
			}

			return directmethod.DirectMethodResponse{
				Status:  200,
				Payload: telemetry,
			}, nil
		},

		// Process data method - accepts JSON array
		"ProcessData": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			appLogger.Information(fmt.Sprintf("ProcessData called with payload: %v", payload))

			data, ok := payload.([]interface{})
			if !ok {
				return directmethod.DirectMethodResponse{
					Status:  400,
					Payload: map[string]string{"error": "Payload should be an array"},
				}, nil
			}

			appLogger.Information(fmt.Sprintf("Processing %d items", len(data)))

			return directmethod.DirectMethodResponse{
				Status: 200,
				Payload: map[string]interface{}{
					"processed": len(data),
					"timestamp": time.Now().Format(time.RFC3339),
				},
			}, nil
		},
	}

	// Step 3: Initialize DirectMethodClient singleton
	appLogger.Information("Initializing DirectMethodClient...")
	client, err := directmethod.GetInstance("myModule", handlers)
	if err != nil {
		log.Fatalf("Failed to create DirectMethodClient: %v", err)
	}

	// Step 4: Create MQTT configuration
	// Default config uses environment variables or defaults
	mqttConfig := models.NewMqttConfigModel()

	// Optionally customize config
	mqttConfig.ServerURIs = []string{"mqtt://localhost:1883"}
	// mqttConfig.Username = "user"
	// mqttConfig.Password = "pass"

	// Step 5: Initialize the client with MQTT connection
	appLogger.Information("Connecting to MQTT broker...")
	if err := client.Init(mqttConfig, appLogger); err != nil {
		log.Fatalf("Failed to initialize DirectMethodClient: %v", err)
	}

	appLogger.Information("DirectMethodClient initialized successfully!")
	appLogger.Information("Listening for direct method calls on: dm/myModule/")
	appLogger.Information("Registered methods:")
	for methodName := range handlers {
		appLogger.Information(fmt.Sprintf("  - %s", methodName))
	}

	// Step 6: Keep the application running
	// Direct method requests will be handled automatically
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Simulate application running for 1 minute
	select {
	case <-time.After(1 * time.Minute):
		appLogger.Information("Demo timeout reached")
	case <-ctx.Done():
		appLogger.Information("Shutdown signal received")
	}

	// Step 7: Cleanup
	appLogger.Information("Shutting down DirectMethodClient...")
	client.Stop()
	appLogger.Information("DirectMethodClient stopped")
}

// ============================================================================
// Additional Examples: Using DirectMethodClient with complex payloads
// ============================================================================

// Example handler with structured payload
func ExampleWithStructuredPayload() {
	// Define a structured request/response type
	type ConfigRequest struct {
		Interval int    `json:"interval"`
		Mode     string `json:"mode"`
		Enabled  bool   `json:"enabled"`
	}

	type ConfigResponse struct {
		Success   bool   `json:"success"`
		Message   string `json:"message"`
		AppliedAt string `json:"appliedAt"`
	}

	appLogger := logger.NewConsoleLogger(logger.Information)

	handlers := map[string]directmethod.DirectMethodHandler{
		"SetConfig": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			// Convert payload to JSON bytes, then unmarshal to struct
			jsonData, err := json.Marshal(payload)
			if err != nil {
				return directmethod.DirectMethodResponse{
					Status:  400,
					Payload: map[string]string{"error": "Invalid JSON"},
				}, nil
			}

			var configReq ConfigRequest
			if err := json.Unmarshal(jsonData, &configReq); err != nil {
				return directmethod.DirectMethodResponse{
					Status:  400,
					Payload: map[string]string{"error": "Failed to parse config"},
				}, nil
			}

			appLogger.Information(fmt.Sprintf(
				"Config applied: interval=%d, mode=%s, enabled=%v",
				configReq.Interval,
				configReq.Mode,
				configReq.Enabled,
			))

			response := ConfigResponse{
				Success:   true,
				Message:   "Configuration applied",
				AppliedAt: time.Now().Format(time.RFC3339),
			}

			return directmethod.DirectMethodResponse{
				Status:  200,
				Payload: response,
			}, nil
		},
	}

	client, _ := directmethod.GetInstance("structuredModule", handlers)
	config := models.NewMqttConfigModel()
	client.Init(config, appLogger)

	defer client.Stop()
}

// Example handler with error handling
func ExampleWithErrorHandling() {
	appLogger := logger.NewConsoleLogger(logger.Information)

	handlers := map[string]directmethod.DirectMethodHandler{
		"LongRunningTask": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			// Simulate a long-running task
			time.Sleep(2 * time.Second)

			// Simulate potential errors
			taskID, ok := payload.(map[string]interface{})["taskId"]
			if !ok {
				return directmethod.DirectMethodResponse{
					Status:  400,
					Payload: map[string]string{"error": "Missing taskId"},
				}, nil
			}

			appLogger.Information(fmt.Sprintf("Task %v completed", taskID))

			return directmethod.DirectMethodResponse{
				Status: 200,
				Payload: map[string]interface{}{
					"taskId":    taskID,
					"status":    "completed",
					"timestamp": time.Now().Format(time.RFC3339),
				},
			}, nil
		},
	}

	client, _ := directmethod.GetInstance("taskModule", handlers)
	config := models.NewMqttConfigModel()
	client.Init(config, appLogger)

	defer client.Stop()
}
