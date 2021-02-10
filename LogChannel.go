package main

import (
	"fmt"
	"time"

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
	DISABLED LogLevel = -1
	// Info lowest LogLevel
	INFO LogLevel = 0
	// Debug LogLevel 1
	DEBUG LogLevel = 1
	// Warning LogLevel 2
	WARNING LogLevel = 2
	// Error LogLevel 3
	ERROR LogLevel = 3
	// Fatal LogLevel 4
	FATAL LogLevel = 4
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

		if log.level >= logChannel.level {
			switch log.level {
			case INFO:
				logChannel.logger.Info(log.message, log.data...)
				break
			case DEBUG:
				logChannel.logger.Debug(log.message, log.data...)
				break
			case WARNING:
				logChannel.logger.Warn(log.message, log.data...)
				break
			case ERROR:
				logChannel.logger.Error(log.message, log.data...)
				break
			case FATAL:
				logChannel.logger.Fatal(log.message, log.data...)
				break
			default:
				logChannel.logger.Error(fmt.Sprintf("MINOSSE: LOG LEVEL %d UNDEFINED", log.level))
				break
			}
		}
	}
}

func (logChannel *LogChannel) error(message string, err error) {
	if logChannel.level != DISABLED {
		logChannel.channel <- Log{
			level:   ERROR,
			message: message,
			data:    []zap.Field{zap.Error(err)},
		}
	}
	return
}

func (LogChannel *LogChannel) logRequest(start time.Time, requestUri, requestMethod *string, statusCode *int, remoteAddr *string) {
	if logChannel.level != DISABLED {
		end := time.Now()
		logChannel.channel <- Log{
			level:   INFO,
			message: "Request received",
			data:    []zap.Field{zap.String("URI", *requestUri), zap.String("Method", *requestMethod), zap.Int("Status", *statusCode), zap.Duration("Duration: ", end.Sub(start)), zap.String("Remote address", *remoteAddr)},
		}
	}
	return
}
