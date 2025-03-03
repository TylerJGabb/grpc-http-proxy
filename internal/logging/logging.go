package logging

import (
	"io"
	"log/slog"
	"os"
)

// structuredLoggingEnvVar logging off by default
var structuredLoggingEnvVar bool = false

// loggingLevelEnvVar default logging level is info
var loggingLevelEnvVar slog.Level = slog.LevelInfo

func init() {
	if os.Getenv("STRUCTURED_LOGGING") == "on" {
		structuredLoggingEnvVar = true
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		loggingLevelEnvVar.UnmarshalText([]byte(logLevel))
	}
}

type LoggerOptions struct {
	requestId   string
	operationId string
	attributes  map[string]string
	output      io.Writer
	structured  bool
	logLevel    slog.Level
}
type LoggerOptionFuction func(*LoggerOptions)

func WithAttribute(key, value string) LoggerOptionFuction {
	return func(l *LoggerOptions) {
		l.attributes[key] = value
	}
}

func WithRequestId(requestId string) LoggerOptionFuction {
	return func(l *LoggerOptions) {
		l.requestId = requestId
	}
}

func WithOperationId(operationId string) LoggerOptionFuction {
	return func(l *LoggerOptions) {
		l.operationId = operationId
	}
}

func WithOutput(w io.Writer) LoggerOptionFuction {
	return func(l *LoggerOptions) {
		l.output = w
	}
}

func WithStructuredLogging(enabled bool) LoggerOptionFuction {
	return func(l *LoggerOptions) {
		l.structured = enabled
	}
}

func WithLogLevel(level slog.Level) LoggerOptionFuction {
	return func(l *LoggerOptions) {
		l.logLevel = level
	}
}

func (l *LoggerOptions) Attrs() []slog.Attr {
	attrs := make([]slog.Attr, 0)
	if l.requestId != "" {
		attrs = append(attrs, slog.Attr{
			Key:   "requestId",
			Value: slog.StringValue(l.requestId),
		})
	}
	if l.operationId != "" {
		attrs = append(attrs, slog.Attr{
			Key:   "operationId",
			Value: slog.StringValue(l.operationId),
		})
	}
	for attributeKey, attributeValue := range l.attributes {
		attrs = append(attrs, slog.Attr{
			Key:   attributeKey,
			Value: slog.StringValue(attributeValue),
		})
	}
	return attrs
}

/*
NewLogger creates a new logger.
By default this logger will output to stdout, in un-structured text format.
If STRUCTURED_LOGGING environment variable is set to "on", output will be written in structured JSON format by deafult.
You can customize the logger by passing LoggerOptionFuction to this function, which will override the default any defaults.
*/
func NewLogger(optFuncs ...LoggerOptionFuction) *slog.Logger {
	loggerOpts := &LoggerOptions{
		output:     os.Stdout,
		structured: structuredLoggingEnvVar,
		logLevel:   loggingLevelEnvVar,
		attributes: make(map[string]string),
	}
	for _, optFunc := range optFuncs {
		optFunc(loggerOpts)
	}
	handler := buildHandler(loggerOpts)
	return slog.New(handler)
}

func buildHandler(loggerOptions *LoggerOptions) slog.Handler {
	handlerOpts := &slog.HandlerOptions{
		Level: loggerOptions.logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "level" {
				return slog.Attr{
					Key:   "severity",
					Value: a.Value,
				}
			}
			if a.Key == "msg" {
				return slog.Attr{
					Key:   "message",
					Value: a.Value,
				}
			} else if a.Key == "time" {
				// Convert time to UTC
				return slog.Attr{
					Key:   "time",
					Value: slog.TimeValue(a.Value.Time().UTC()),
				}
			}
			return a
		},
	}
	var handler slog.Handler
	if loggerOptions.structured {
		handler = slog.NewJSONHandler(loggerOptions.output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(loggerOptions.output, handlerOpts)
	}
	return handler.WithAttrs(loggerOptions.Attrs())
}
