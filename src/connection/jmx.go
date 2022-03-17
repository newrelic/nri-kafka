package connection

import (
	"context"
	"errors"
	"fmt"
	"github.com/newrelic/nrjmx/gojmx"
	"golang.org/x/sync/semaphore"
)

var (
	// ErrJMXConnection happens when the configuration provided fails to open a new JMX connection.
	ErrJMXConnection = errors.New("JMX connection failed")
	// ErrJMXCollection happens when the JMX connection can be established but querying the data fails.
	ErrJMXCollection = errors.New("JMX collection failed")
)

// JMXProvider interface to open new JMX connections.
type JMXProvider interface {
	NewConnection(config *gojmx.JMXConfig) (conn JMXConnection, err error)
}

// JMXProviderWithConnectionsLimit will be able to allocate new JMX connections, as long as maxConnections is not reached.
type JMXProviderWithConnectionsLimit struct {
	ctx context.Context
	sem *semaphore.Weighted
}

// NewJMXProviderWithLimit creates a new instance of JMXProvider.
func NewJMXProviderWithLimit(ctx context.Context, maxConnections int) JMXProvider {
	return &JMXProviderWithConnectionsLimit{
		ctx: ctx,
		sem: semaphore.NewWeighted(int64(maxConnections)),
	}
}

// NewConnection will return a new JMX connection if the maximum number of concurrent connections
// is not reached, otherwise will block until a connection is released.
func (p *JMXProviderWithConnectionsLimit) NewConnection(config *gojmx.JMXConfig) (JMXConnection, error) {
	err := p.sem.Acquire(p.ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to open new connection, %w", err)
	}

	client, err := gojmx.NewClient(p.ctx).Open(config)
	if err != nil {
		// In case of error, we unlock a new connection.
		p.sem.Release(1)
		return nil, err
	}

	return &jmxConnection{
		Client: client,
		sem:    p.sem,
	}, nil
}

// JMXConnection interface for JMX connection.
type JMXConnection interface {
	QueryMBeanAttributes(mBeanNamePattern string) ([]*gojmx.AttributeResponse, error)
	Close() error
}

// jmxConnection is a wrapper over gojmx.Client to include the semaphore that will
// limit the amount of concurrent connections.
type jmxConnection struct {
	*gojmx.Client
	sem *semaphore.Weighted
}

func (j *jmxConnection) QueryMBeanAttributes(mBeanNamePattern string) ([]*gojmx.AttributeResponse, error) {
	return j.Client.QueryMBeanAttributes(mBeanNamePattern)
}

func (j *jmxConnection) Close() error {
	// In case of error on closing the connection, gojmx will kill the subprocess, then we do the release anyway.
	defer j.sem.Release(1)
	return j.Client.Close()
}
