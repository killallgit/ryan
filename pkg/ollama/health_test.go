package ollama

import (
	"context"
	"testing"
	"time"
)

func TestCheckHealthWithTimeout(t *testing.T) {
	// Test with a non-existent server (should fail quickly)
	client := NewClient("http://localhost:99999") // Unlikely to be running

	health, err := client.CheckHealthWithTimeout(1 * time.Second)
	if err != nil {
		t.Errorf("CheckHealthWithTimeout should not return an error, got: %v", err)
	}

	if health.Available {
		t.Error("Health should show as not available for non-existent server")
	}

	if health.Error == nil {
		t.Error("Health should have an error for non-existent server")
	}
}

func TestCheckModelWithTimeout(t *testing.T) {
	// Test with a non-existent server (should fail quickly)
	client := NewClient("http://localhost:99999") // Unlikely to be running

	hasModel, err := client.CheckModelWithTimeout("test-model", 1*time.Second)
	if err == nil {
		t.Error("CheckModelWithTimeout should return an error for non-existent server")
	}

	if hasModel {
		t.Error("Should not report model as available for non-existent server")
	}
}

func TestCheckHealth(t *testing.T) {
	// Test with a non-existent server using context
	client := NewClient("http://localhost:99999")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	health, err := client.CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth should not return an error, got: %v", err)
	}

	if health.Available {
		t.Error("Health should show as not available for non-existent server")
	}

	if len(health.Models) > 0 {
		t.Error("Should not have any models for non-existent server")
	}
}
