package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
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
	// INFO lowest LogLevel
	INFO LogLevel = 0
	// DEBUG LogLevel 1
	DEBUG LogLevel = 1
	// WARNING LogLevel 2
	WARNING LogLevel = 2
	// ERROR LogLevel 3
	ERROR LogLevel = 3
	// FATAL LogLevel 4
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
				color.Set(color.FgYellow)
				logChannel.logger.Warn(log.message, log.data...)
				color.Unset()
				break
			case ERROR:
				color.Set(color.FgRed)
				logChannel.logger.Error(log.message, log.data...)
				color.Unset()
				break
			case FATAL:
				color.Set(color.FgHiRed)
				logChannel.logger.Fatal(log.message, log.data...)
				color.Unset()
				break
			default:
				logChannel.logger.Error(fmt.Sprintf("MINOSSE: LOG LEVEL %d UNDEFINED", log.level))
				break
			}
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

func (logChannel *LogChannel) logRequest(start time.Time, requestUri, requestMethod *string, statusCode *int, remoteAddr *string, transportProtocol *string) {
	end := time.Now()
	logChannel.channel <- Log{
		level:   INFO,
		message: "Request received",
		data:    []zap.Field{zap.String("URI", *requestUri), zap.String("Method", *requestMethod), zap.Int("Status", *statusCode), zap.Duration("Duration: ", end.Sub(start)), zap.String("Remote address", *remoteAddr), zap.String("Transport protocol", *transportProtocol)},
	}
}

func (logChannel *LogChannel) logWholeRequest(request *http.Request, response *Response, start *time.Time) {
	if request == nil {
		logChannel.channel <- Log{level: ERROR, message: "Nil request"}
		return
	}
	if response == nil {
		logChannel.channel <- Log{level: ERROR, message: "Nil response"}
		return
	}
	if response.statusCode == 0 {
		panic(response)
	}

	end := time.Now()
	var sb strings.Builder
	var body []byte

	_, err := request.Body.Read(body)
	if err != nil && err != io.EOF {
		logChannel.error("Error while reading request body", err)
	}

	for key, val := range request.Header {
		sb.WriteString(key)
		sb.WriteString(": ")
		for _, headerVal := range val {
			sb.WriteString(headerVal)
		}
		sb.WriteString(", ")
	}

	logChannel.channel <- Log{
		level:   INFO,
		message: ">>>>",
		data: []zap.Field{
			zap.Int("response_code", response.statusCode),
			zap.String("response_status", response.status),
			zap.String("request_method", request.Method),
			zap.String("request_uri", request.URL.String()),
			zap.String("request_headers", sb.String()),
			zap.String("request_body", string(body)),
			zap.String("request_remote_address", request.RemoteAddr),
			zap.Duration("duration", end.Sub(*start)),
		},
	}
}
