package tests

import (
	"go.uber.org/mock/gomock"
)

// MockTestConfig holds mock dependencies for testing
type MockTestConfig struct {
	Controller *gomock.Controller
}

// MockController interface that both testing.T and GinkgoT satisfy
type MockController interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Helper()
}

// SetupMockTest creates a new mock test configuration
func SetupMockTest(t MockController) *MockTestConfig {
	ctrl := gomock.NewController(t)
	return &MockTestConfig{
		Controller: ctrl,
	}
}

// TeardownMockTest cleans up the mock test configuration
func (mtc *MockTestConfig) TeardownMockTest() {
	if mtc.Controller != nil {
		mtc.Controller.Finish()
	}
}
