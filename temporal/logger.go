package temporal

import (
	"github.com/rs/zerolog"
	temporallog "go.temporal.io/sdk/log"
)

type ZerologAdapter struct {
	logger zerolog.Logger
}

// NewZerologAdapter creates a new ZerologAdapter
func NewZerologAdapter(logger zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{
		logger: logger,
	}
}

func (z *ZerologAdapter) Debug(msg string, keyvals ...interface{}) {
	z.log(z.logger.Debug(), msg, keyvals...)
}

func (z *ZerologAdapter) Info(msg string, keyvals ...interface{}) {
	z.log(z.logger.Info(), msg, keyvals...)
}

func (z *ZerologAdapter) Warn(msg string, keyvals ...interface{}) {
	z.log(z.logger.Warn(), msg, keyvals...)
}

func (z *ZerologAdapter) Error(msg string, keyvals ...interface{}) {
	z.log(z.logger.Error(), msg, keyvals...)
}

func (z *ZerologAdapter) log(event *zerolog.Event, msg string, keyvals ...interface{}) {
	// Process key-value pairs
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key, ok := keyvals[i].(string)
			if !ok {
				key = "unknown"
			}
			event = event.Interface(key, keyvals[i+1])
		} else {
			// Handle odd number of keyvals
			event = event.Interface("unknown", keyvals[i])
		}
	}
	event.Msg(msg)
}

func (z *ZerologAdapter) WithCallerSkip(skip int) temporallog.Logger {
	// Create a new logger with the caller skip
	newLogger := z.logger.With().CallerWithSkipFrameCount(skip + 2).Logger()
	return NewZerologAdapter(newLogger)
}

func (z *ZerologAdapter) With(keyvals ...interface{}) temporallog.Logger {
	ctx := z.logger.With()

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key, ok := keyvals[i].(string)
			if !ok {
				key = "unknown"
			}
			ctx = ctx.Interface(key, keyvals[i+1])
		} else {
			// Handle odd number of keyvals
			ctx = ctx.Interface("unknown", keyvals[i])
		}
	}

	return NewZerologAdapter(ctx.Logger())
}
