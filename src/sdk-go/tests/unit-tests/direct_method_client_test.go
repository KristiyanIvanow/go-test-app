package unit_tests_test

import (
	"testing"

	"github.com/KristiyanIvanow/go-test-app/src/sdk-go/src/directmethod"
)

func TestDirectMethodGetInstanceRejectsEmptyModuleID(t *testing.T) {
	if _, err := directmethod.GetInstance("", map[string]directmethod.DirectMethodHandler{}); err == nil {
		t.Fatal("expected error for empty moduleID")
	}
}

func TestDirectMethodGetInstanceRejectsNilHandlers(t *testing.T) {
	if _, err := directmethod.GetInstance("mod", nil); err == nil {
		t.Fatal("expected error for nil handlers map")
	}
}

func TestDirectMethodGetInstanceIsSingleton(t *testing.T) {
	handlers := map[string]directmethod.DirectMethodHandler{
		"Ping": func(payload interface{}) (directmethod.DirectMethodResponse, error) {
			return directmethod.DirectMethodResponse{Status: 200, Payload: "pong"}, nil
		},
	}
	a, err := directmethod.GetInstance("mod", handlers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := directmethod.GetInstance("mod", handlers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != b {
		t.Error("expected same singleton instance")
	}
}

func TestDirectMethodResponseShape(t *testing.T) {
	r := directmethod.DirectMethodResponse{
		Status:  200,
		Payload: map[string]string{"ok": "true"},
	}
	if r.Status != 200 {
		t.Errorf("Status = %d, want 200", r.Status)
	}
}
