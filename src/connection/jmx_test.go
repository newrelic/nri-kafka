package connection

import (
	"context"
	"testing"

	"github.com/newrelic/nrjmx/gojmx"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

func Test_NewConnection(t *testing.T) {
	// GIVEN a limit of 3 concurrent connections
	var maxConnections int64 = 3
	ctx := context.Background()
	mockProvider := &JMXProviderWithConnectionsLimit{
		ctx: ctx,
		sem: semaphore.NewWeighted(maxConnections),
		jmxOpenFn: func(*gojmx.JMXConfig) (*gojmx.Client, error) {
			return gojmx.NewClient(ctx), nil
		},
	}

	// WHEN a connection fail
	client, err := mockProvider.NewConnection(nil)
	assert.NotNil(t, client)
	assert.NoError(t, err)

	// THEN we can open just 2 concurrent connections
	assert.False(t, mockProvider.sem.TryAcquire(maxConnections))
	assert.True(t, mockProvider.sem.TryAcquire(maxConnections-1))

	// WHEN one connection is closed
	client.Close()

	// THEN we can open just 1 connection
	assert.False(t, mockProvider.sem.TryAcquire(2))
	assert.True(t, mockProvider.sem.TryAcquire(1))
}

func Test_NewConnection_OnError(t *testing.T) {
	// GIVEN a limit of 3 concurrent connections
	var maxConnections int64 = 3
	mockProvider := &JMXProviderWithConnectionsLimit{
		ctx: context.Background(),
		sem: semaphore.NewWeighted(maxConnections),
		jmxOpenFn: func(*gojmx.JMXConfig) (*gojmx.Client, error) {
			return nil, ErrJMXConnection
		},
	}

	// WHEN a connection fail
	client, err := mockProvider.NewConnection(nil)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrJMXConnection)

	// THEN we can open 3 concurrent connections
	assert.True(t, mockProvider.sem.TryAcquire(maxConnections))
}
