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

func (z *ZerologAdapter) Debug(msg string, keyvals ...any) {
	z.log(z.logger.Debug(), msg, keyvals...)
}

func (z *ZerologAdapter) Info(msg string, keyvals ...any) {
	z.log(z.logger.Info(), msg, keyvals...)
}

func (z *ZerologAdapter) Warn(msg string, keyvals ...any) {
	z.log(z.logger.Warn(), msg, keyvals...)
}

func (z *ZerologAdapter) Error(msg string, keyvals ...any) {
	z.log(z.logger.Error(), msg, keyvals...)
}

func (z *ZerologAdapter) log(event *zerolog.Event, msg string, keyvals ...any) {
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
	// zerolog resolves the caller inside Event.Msg. From the caller's log
	// call site the stack adds four frames before zerolog reads the PC:
	// ZerologAdapter.Info/Debug/Warn/Error -> ZerologAdapter.log -> Event.Msg
	// -> zerolog's caller hook, on top of zerolog's own 2-frame base. Hence
	// skip+4 (empirically verified against the SDK's caller reporting).
	newLogger := z.logger.With().CallerWithSkipFrameCount(skip + 4).Logger()
	return NewZerologAdapter(newLogger)
}

func (z *ZerologAdapter) With(keyvals ...any) temporallog.Logger {
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
