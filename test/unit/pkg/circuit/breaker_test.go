package circuit

import (
	"context"
	"errors"
	"testing"
	"time"

	"odin/pkg/circuit"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_Execute(t *testing.T) {
	config := circuit.Config{
		MaxRequests:  5,
		Interval:     time.Second,
		Timeout:      time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	}

	cb := circuit.NewCircuitBreaker("test", config, logrus.New())

	// Test successful execution
	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, circuit.StateClosed, cb.State())
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := circuit.Config{
		MaxRequests:  2,
		Interval:     100 * time.Millisecond,
		Timeout:      100 * time.Millisecond,
		FailureRatio: 0.5,
		MinRequests:  2,
	}

	cb := circuit.NewCircuitBreaker("test", config, logrus.New())

	// Initial state should be closed
	assert.Equal(t, circuit.StateClosed, cb.State())

	// Execute failing requests to trigger state change
	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
	}

	// Circuit should be open after failures
	assert.Equal(t, circuit.StateOpen, cb.State())

	// Immediate request should fail with circuit open error
	_, err := cb.Execute(func() (interface{}, error) {
		return "should not execute", nil
	})
	assert.Equal(t, circuit.ErrCircuitOpen, err)

	// Wait for timeout to pass
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open
	_, err = cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, circuit.StateClosed, cb.State())
}

func TestCircuitBreaker_ExecuteWithContext(t *testing.T) {
	config := circuit.Config{
		MaxRequests:  5,
		Interval:     time.Second,
		Timeout:      time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	}

	cb := circuit.NewCircuitBreaker("test", config, logrus.New())
	ctx := context.Background()

	result, err := cb.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		return "context success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "context success", result)
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := circuit.NewManager(logrus.New())

	config := circuit.Config{
		MaxRequests:  5,
		Interval:     time.Second,
		Timeout:      time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	}

	// Get breaker should create new one
	cb1 := manager.GetBreaker("service1", config)
	assert.NotNil(t, cb1)

	// Getting same breaker should return existing one
	cb2 := manager.GetBreaker("service1", config)
	assert.Same(t, cb1, cb2)

	// Get status should return info for all breakers
	status := manager.GetBreakerStatus()
	assert.Contains(t, status, "service1")
}
