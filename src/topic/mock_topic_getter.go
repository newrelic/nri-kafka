package topic

import (
	"github.com/stretchr/testify/mock"
)

// MockConnection implements Connection to facilitate testing.
type MockGetter struct {
	mock.Mock
}

// Topics provides a mock function with given fields:
func (_m *MockGetter) Topics() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
