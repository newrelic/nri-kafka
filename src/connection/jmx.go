package connection

import (
	"context"
	"errors"
	"fmt"

	"github.com/newrelic/nri-kafka/src/args"
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
	ctx       context.Context
	sem       *semaphore.Weighted
	jmxOpenFn func(*gojmx.JMXConfig) (*gojmx.Client, error)
}

// NewJMXProviderWithLimit creates a new instance of JMXProvider.
func NewJMXProviderWithLimit(ctx context.Context, maxConnections int) JMXProvider {
	return &JMXProviderWithConnectionsLimit{
		ctx: ctx,
		sem: semaphore.NewWeighted(int64(maxConnections)),
		jmxOpenFn: func(config *gojmx.JMXConfig) (*gojmx.Client, error) {
			return gojmx.NewClient(ctx).Open(config)
		},
	}
}

// NewConnection will return a new JMX connection if the maximum number of concurrent connections
// is not reached, otherwise will block until a connection is released.
func (p *JMXProviderWithConnectionsLimit) NewConnection(config *gojmx.JMXConfig) (JMXConnection, error) {
	err := p.sem.Acquire(p.ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to open new connection, %w", err)
	}

	client, err := p.jmxOpenFn(config)
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
	QueryMBeanNames(mBeanPattern string) ([]string, error)
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

func (j *jmxConnection) QueryMBeanNames(mBeanPattern string) ([]string, error) {
	return j.Client.QueryMBeanNames(mBeanPattern)
}

func (j *jmxConnection) Close() error {
	// In case of error on closing the connection, gojmx will kill the subprocess, then we do the release anyway.
	defer j.sem.Release(1)
	return j.Client.Close()
}

// ConfigBuilder will be used to build the JMX connection config.
type ConfigBuilder struct {
	config *gojmx.JMXConfig
}

// NewConfigBuilder creates a new instance of a ConfigBuilder.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &gojmx.JMXConfig{},
	}
}

// FromArgs will extract configuration from global arguments.
func (cb *ConfigBuilder) FromArgs() *ConfigBuilder {
	cb.config.Username = args.GlobalArgs.DefaultJMXUser
	cb.config.Password = args.GlobalArgs.DefaultJMXPassword
	cb.config.RequestTimeoutMs = int64(args.GlobalArgs.Timeout)
	if args.GlobalArgs.KeyStore != "" && args.GlobalArgs.KeyStorePassword != "" && args.GlobalArgs.TrustStore != "" && args.GlobalArgs.TrustStorePassword != "" {
		cb.config.KeyStore = args.GlobalArgs.KeyStore
		cb.config.KeyStorePassword = args.GlobalArgs.KeyStorePassword
		cb.config.TrustStore = args.GlobalArgs.TrustStore
		cb.config.TrustStorePassword = args.GlobalArgs.TrustStorePassword
	}
	return cb
}

// WithHostname will add the hostname to jmx config.
func (cb *ConfigBuilder) WithHostname(hostname string) *ConfigBuilder {
	cb.config.Hostname = hostname
	return cb
}

// WithPort will add the port to jmx config.
func (cb *ConfigBuilder) WithPort(port int) *ConfigBuilder {
	cb.config.Port = int32(port)
	return cb
}

// WithUsername will add the user to jmx config.
func (cb *ConfigBuilder) WithUsername(user string) *ConfigBuilder {
	cb.config.Username = user
	return cb
}

// WithPassword will add the password to jmx config.
func (cb *ConfigBuilder) WithPassword(password string) *ConfigBuilder {
	cb.config.Password = password
	return cb
}

// WithJMXHostSettings is a helper to set all attributes from the provided JMXHost.
func (cb *ConfigBuilder) WithJMXHostSettings(jmxInfo *args.JMXHost) *ConfigBuilder {
	return cb.
		WithHostname(jmxInfo.Host).WithPort(jmxInfo.Port).
		WithUsername(jmxInfo.User).WithPassword(jmxInfo.Password)
}

// Build returns the jmx config.
func (cb *ConfigBuilder) Build() *gojmx.JMXConfig {
	return cb.config
}
