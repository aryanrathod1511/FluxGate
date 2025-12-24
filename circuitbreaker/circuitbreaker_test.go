package circuitbreaker

import (
	"FluxGate/configuration"
	"testing"
	"time"
)

func TestCircuitBreakerTransitions(t *testing.T) {
	cb := New(configuration.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		WindowSeconds:    60,
		OpenSeconds:      1,
		HalfOpenRequests: 1,
		SuccessThreshold: 1,
	})

	// Initially closed
	if !cb.Allow() {
		t.Fatalf("expected allow in closed state")
	}

	// Two failures should open the breaker
	cb.OnFailure()
	cb.OnFailure()
	if cb.Allow() {
		t.Fatalf("expected breaker to be open after failures")
	}

	// Wait for open timeout to expire, should move to half-open on next Allow
	time.Sleep(1100 * time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("expected half-open to allow a trial request")
	}

	// Success in half-open should close the breaker
	cb.OnSuccess()
	if !cb.Allow() {
		t.Fatalf("expected closed state after successful half-open trial")
	}
}
