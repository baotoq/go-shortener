package usecase

import (
	"context"

	dapr "github.com/dapr/go-sdk/client"
)

// DaprClient is a minimal interface wrapping the Dapr client methods actually used by URLService.
// This allows for easier mocking in tests without implementing the entire dapr.Client interface.
type DaprClient interface {
	InvokeMethod(ctx context.Context, appID, methodName, verb string) ([]byte, error)
	PublishEvent(ctx context.Context, pubsubName, topicName string, data interface{}, opts ...dapr.PublishEventOption) error
	Close()
}
