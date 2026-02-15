package testutil

import (
	"context"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/stretchr/testify/mock"
)

// MockDaprClient is a testify mock for dapr.Client.
// It only provides mock implementations for the methods actually used by the codebase:
// InvokeMethod, PublishEvent, and Close
//
// Note: This does not implement the full dapr.Client interface.
// It is sufficient for unit testing the URLService and similar code paths.
type MockDaprClient struct {
	mock.Mock
}

func (m *MockDaprClient) InvokeMethod(ctx context.Context, appID, methodName, verb string) ([]byte, error) {
	args := m.Called(ctx, appID, methodName, verb)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockDaprClient) PublishEvent(ctx context.Context, pubsubName, topicName string, data interface{}, opts ...dapr.PublishEventOption) error {
	args := m.Called(ctx, pubsubName, topicName, data)
	return args.Error(0)
}

func (m *MockDaprClient) Close() {
	m.Called()
}
