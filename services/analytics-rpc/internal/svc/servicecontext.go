package svc

import "go-shortener/services/analytics-rpc/internal/config"

type ServiceContext struct {
	Config config.Config
	// Phase 8 adds: ClickModel model.ClickModel
	// Phase 9 adds: KafkaConsumer (configured elsewhere)
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
