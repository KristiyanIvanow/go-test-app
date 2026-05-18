package unit_tests_test

import (
	"testing"

	"github.com/KristiyanIvanow/go-test-app/src/containerproperties"
)

func TestContainerPropertiesGetInstanceRejectsEmptyModuleID(t *testing.T) {
	_, err := containerproperties.GetInstance("", nil)
	if err == nil {
		t.Fatal("expected error for empty moduleID")
	}
}

func TestContainerPropertiesGetInstanceIsSingleton(t *testing.T) {
	a, err := containerproperties.GetInstance("module01", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := containerproperties.GetInstance("module01", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != b {
		t.Errorf("expected same singleton instance")
	}
}

func TestSetDesiredPropertyUpdateCallbackRejectsNil(t *testing.T) {
	c, err := containerproperties.GetInstance("module01", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.SetDesiredPropertyUpdateCallback(nil); err == nil {
		t.Errorf("expected error when registering nil handler")
	}
}

func TestUpdateReportedPropertiesRejectsNilMap(t *testing.T) {
	c, err := containerproperties.GetInstance("module01", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.UpdateReportedProperties(nil); err == nil {
		t.Errorf("expected error for nil reported properties")
	}
}
