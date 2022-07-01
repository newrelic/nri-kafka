package mocks

import (
	"errors"
	"fmt"

	"github.com/newrelic/nri-kafka/src/connection"
	"github.com/newrelic/nrjmx/gojmx"
)

var (
	ErrQuery = errors.New("query failed")
)

type MockJMXResponse struct {
	Result []*gojmx.AttributeResponse
	Err    error
}

type MockJMXProvider struct {
	Response         *MockJMXResponse
	Names            []string
	MBeanNamePattern string
}

func NewEmptyMockJMXProvider() *MockJMXProvider {
	return &MockJMXProvider{
		Response: &MockJMXResponse{
			Result: []*gojmx.AttributeResponse{},
		},
	}
}

func (m *MockJMXProvider) QueryMBeanAttributes(mBeanNamePattern string) ([]*gojmx.AttributeResponse, error) {
	if m.MBeanNamePattern != "" && m.MBeanNamePattern != mBeanNamePattern {
		return nil, fmt.Errorf("%w: expected bean '%s' got '%s'", ErrQuery, m.MBeanNamePattern, mBeanNamePattern)
	}
	return m.Response.Result, m.Response.Err
}

func (m *MockJMXProvider) QueryMBeanNames(mBeanNamePattern string) ([]string, error) {
	if m.MBeanNamePattern != "" && m.MBeanNamePattern != mBeanNamePattern {
		return nil, fmt.Errorf("%w: expected bean pattern '%s' got '%s'", ErrQuery, m.MBeanNamePattern, mBeanNamePattern)
	}
	return m.Names, nil
}

func (m *MockJMXProvider) Close() error {
	return m.Response.Err
}

func (m *MockJMXProvider) NewConnection(config *gojmx.JMXConfig) (conn connection.JMXConnection, err error) {
	return m, m.Response.Err
}
