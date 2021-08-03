package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// LogChannel A logging channel. Consists of a zap logger, a Log channel and a preconfigured LogLevel, which is used for filtering unwanted logs
type LogChannel struct {
	logger  zap.Logger
	channel chan Log
	level   LogLevel
}

// Log Main log structure. Consists of a LogLevel, the message and the array of zap.Fields data to be logged
type Log struct {
	level   LogLevel
	message string
	data    []zap.Field
}

// LogLevel a numeric log level used to discriminate between necessary and unnecessary logs
type LogLevel int8

const (
	DISABLED LogLevel = 100
	// DEBUG lowest LogLevel
	DEBUG LogLevel = iota
	// INFO LogLevel 1
	INFO
	// WARNING LogLevel 2
	WARNING
	// ERROR LogLevel 3
	ERROR
	// DPANIC LogLevel 4
	DPANIC
	// PANIC LogLevel 5
	PANIC
	// FATAL LogLevel 6
	FATAL
)

func newLogChannel(logger *zap.Logger, config *Config) LogChannel {
	return LogChannel{
		logger:  *logger,
		channel: make(chan Log),
		level:   config.Minosse.Log,
	}
}

func (logChannel LogChannel) handleLog() {
	defer logChannel.logger.Sync()

	for {
		log := <-logChannel.channel

		switch log.level {
		case DEBUG:
			logChannel.logger.Debug(log.message, log.data...)
		case INFO:
			logChannel.logger.Info(log.message, log.data...)
		case WARNING:
			color.Set(color.FgYellow)
			logChannel.logger.Warn(log.message, log.data...)
			color.Unset()
		case ERROR:
			color.Set(color.FgRed)
			logChannel.logger.Error(log.message, log.data...)
			color.Unset()
		case FATAL:
			color.Set(color.FgHiRed)
			logChannel.logger.Fatal(log.message, log.data...)
			color.Unset()
		default:
			logChannel.logger.Error(fmt.Sprintf("MINOSSE: LOG LEVEL %d UNDEFINED", log.level))
		}
	}
}

func (logChannel *LogChannel) fatalError(message string, err error) {
	logChannel.channel <- Log{
		level:   FATAL,
		message: message,
		data:    []zap.Field{zap.Error(err)},
	}
}

func (logChannel *LogChannel) error(message string, err error) {
	logChannel.channel <- Log{
		level:   ERROR,
		message: message,
		data:    []zap.Field{zap.Error(err)},
	}
}

func (logChannel *LogChannel) logWholeRequest(ctx *fasthttp.RequestCtx, start *time.Time) {
	if ctx.Response.StatusCode() == 0 {
		panic(ctx)
	}

	end := time.Now()

	logChannel.channel <- Log{
		level:   INFO,
		message: ">>>>",
		data: []zap.Field{
			zap.Int("response_code", ctx.Response.StatusCode()),
			// zap.String("request_method", request.URI()),
			zap.String("request_uri", ctx.Request.URI().String()),
			zap.ByteString("request_headers", ctx.Request.Header.RawHeaders()),
			zap.ByteString("request_body", ctx.PostBody()),
			zap.ByteString("request_remote_address", ctx.Request.Host()),
			zap.Duration("duration", end.Sub(*start)),
		},
	}
}
