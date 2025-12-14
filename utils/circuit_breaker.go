package utils

import (
	"FluxGate/circuitbreaker"
	"net/http"
)

func UpdateCircuitBreaker(cb *circuitbreaker.CircuitBreaker, statusCode int) {
	if cb == nil {
		return
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	if statusCode >= 500 {
		cb.OnFailure()
	} else {
		cb.OnSuccess()
	}
}
