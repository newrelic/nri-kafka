package connection

import (
	"context"
	"errors"
	"fmt"
	"github.com/newrelic/nri-kafka/src/args"
	"golang.org/x/sync/semaphore"
	"sync"

	"github.com/newrelic/nrjmx/gojmx"
)

var (
	ErrJMXCollection = errors.New("JMX collection failed")
	ErrConnectionErr = errors.New("JMX connection failed")

	jmxProvider JMXProvider
	once        sync.Once
)

type JMXProvider interface {
	NewConnection(config *gojmx.JMXConfig) (conn JMXConnection, err error)
}

// JMXProviderWithLimit will be able to allocate new JMX connections, as long as maxConnections is not reached.
type JMXProviderWithLimit struct {
	ctx context.Context
	sem *semaphore.Weighted
}

func GetJMXConnectionProvider() JMXProvider {
	once.Do(func() {
		jmxProvider = newJMXProviderWithLimit(context.Background(), args.GlobalArgs.MaxJMXConnections)
	})
	return jmxProvider
}

func SetJMXConnectionProvider(provider JMXProvider) {
	once.Do(func() {})
	jmxProvider = provider
}

// newJMXProviderWithLimit creates a new instance of JMXProvider.
func newJMXProviderWithLimit(ctx context.Context, maxConnections int) JMXProvider {
	return &JMXProviderWithLimit{
		ctx: ctx,
		sem: semaphore.NewWeighted(int64(maxConnections)),
	}
}

// NewConnection will return a new JMX connection if the maximum number of concurrent connections
// is not reached, otherwise will block until a connection is released.
func (p *JMXProviderWithLimit) NewConnection(config *gojmx.JMXConfig) (conn JMXConnection, err error) {
	err = p.sem.Acquire(p.ctx, 1)
	if err != nil {
		err = fmt.Errorf("failed to open new connection, %w", err)
		return
	}

	defer func() {
		if err != nil {
			p.sem.Release(1)
		}
	}()

	var client *gojmx.Client

	client, err = gojmx.NewClient(p.ctx).Open(config)
	if err != nil {
		return
	}

	conn = &jmxConnection{
		Client: client,
		sem:    p.sem,
	}
	return
}

// JMXConnection interface for JMX connection.
type JMXConnection interface {
	QueryMBeanAttributes(mBeanNamePattern string) ([]*gojmx.AttributeResponse, error)
	Close() error
}

// jmxConnection is a wrapper over jmxClient to include the semaphore that will limit the amount of concurrent
// connections.
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
