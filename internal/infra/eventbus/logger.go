package eventbus

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/go-kratos/kratos/v2/log"
)

// KratosLoggerAdapter adapts Kratos logger to Watermill's LoggerAdapter.
type KratosLoggerAdapter struct {
	logger *log.Helper
	fields watermill.LogFields
}

// NewKratosLoggerAdapter creates a new Watermill logger adapter.
func NewKratosLoggerAdapter(logger log.Logger) watermill.LoggerAdapter {
	return &KratosLoggerAdapter{
		logger: log.NewHelper(logger),
		fields: make(watermill.LogFields),
	}
}

func (l *KratosLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	keyvals := l.toKeyvals(fields, err)
	l.logger.Log(log.LevelError, append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *KratosLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	keyvals := l.toKeyvals(fields, nil)
	l.logger.Log(log.LevelInfo, append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *KratosLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	keyvals := l.toKeyvals(fields, nil)
	l.logger.Log(log.LevelDebug, append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *KratosLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	keyvals := l.toKeyvals(fields, nil)
	l.logger.Log(log.LevelDebug, append([]interface{}{"msg", msg}, keyvals...)...)
}

func (l *KratosLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	newFields := make(watermill.LogFields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &KratosLoggerAdapter{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *KratosLoggerAdapter) toKeyvals(fields watermill.LogFields, err error) []interface{} {
	keyvals := make([]interface{}, 0, (len(l.fields)+len(fields))*2+2)

	for k, v := range l.fields {
		keyvals = append(keyvals, k, v)
	}
	for k, v := range fields {
		keyvals = append(keyvals, k, v)
	}
	if err != nil {
		keyvals = append(keyvals, "error", err)
	}

	return keyvals
}
