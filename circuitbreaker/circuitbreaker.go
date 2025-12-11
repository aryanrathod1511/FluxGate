package circuitbreaker

import (
	"FluxGate/configuration"
	"sync"
	"time"
)

// type CircuitBreaker interface {
// 	Allow() bool
// 	OnSuccess()
// 	OnFailure()
// }

type CircuitBreaker struct {
	mu sync.Mutex

	state int // 0: "Closed", 1: "Open", 2: "HalfOpen"

	// counters
	failures        int
	successes       int
	trialsInFlight  int
	lastFailureTime time.Time

	// config
	failureThreshold int
	window           time.Duration
	openTimeout      time.Duration
	halfOpenLimit    int
	successThreshold int

	openUntil time.Time
}

func New(cfg configuration.CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:            0,
		failureThreshold: cfg.FailureThreshold,
		window:           time.Duration(cfg.WindowSeconds) * time.Second,
		openTimeout:      time.Duration(cfg.OpenSeconds) * time.Second,
		halfOpenLimit:    cfg.HalfOpenRequests,
		successThreshold: cfg.SuccessThreshold,

		failures:       0,
		successes:      0,
		trialsInFlight: 0,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case 1: //open
		if now.After(cb.openUntil) {
			//move to half open
			cb.state = 2 // half open
			cb.successes = 0
			cb.failures = 0
			cb.trialsInFlight = 0
		} else {
			return false
		}
	case 0: //closed
		if !cb.lastFailureTime.IsZero() && now.Sub(cb.lastFailureTime) > cb.window {
			cb.failures = 0
		}
		return true

	case 2: //half open
		if cb.trialsInFlight >= cb.halfOpenLimit {
			return false
		}

		cb.trialsInFlight++
		return true
	}
	return true
}

func (cb *CircuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == 2 { // half open
		cb.successes++
		cb.trialsInFlight--

		if cb.successes >= cb.successThreshold {
			//back to normal
			cb.state = 0 //closed
			cb.failures = 0
			cb.successes = 0
			cb.trialsInFlight = 0
		}
		return
	}

	//closed state
	//Eat five star and do nothing :)
}

func (cb *CircuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.lastFailureTime = now

	//if in hald open : -> open
	if cb.state == 2 {
		cb.state = 1
		cb.openUntil = now.Add(cb.openTimeout)
		cb.failures = 0
		cb.successes = 0
		cb.trialsInFlight = 0
		return
	} else {
		//closed state
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			cb.state = 1 //open
			cb.openUntil = now.Add(cb.openTimeout)
		}

	}
}
