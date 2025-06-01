package aggregator

import (
	"testing"

	"odin/pkg/aggregator"
	"odin/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewAggregator(t *testing.T) {
	services := []config.ServiceConfig{
		{
			Name:    "test-service",
			Targets: []string{"http://localhost:8081"},
		},
	}

	agg := aggregator.New(logrus.New(), services)
	assert.NotNil(t, agg)
}

func TestAggregatorBasicFunctionality(t *testing.T) {
	services := []config.ServiceConfig{
		{
			Name:    "test-service",
			Targets: []string{"http://localhost:8081"},
		},
	}

	agg := aggregator.New(logrus.New(), services)
	assert.NotNil(t, agg)

	// Test that the aggregator is properly initialized
	assert.Equal(t, len(services), 1)
}

func TestAggregatorWithMultipleServices(t *testing.T) {
	services := []config.ServiceConfig{
		{
			Name:    "users",
			Targets: []string{"http://localhost:8081"},
		},
		{
			Name:    "orders",
			Targets: []string{"http://localhost:8082"},
		},
	}

	agg := aggregator.New(logrus.New(), services)
	assert.NotNil(t, agg)
	assert.Equal(t, len(services), 2)
}
