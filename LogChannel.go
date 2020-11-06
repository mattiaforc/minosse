package main

import (
	"fmt"

	"go.uber.org/zap"
)

type LogChannel struct {
	logger zap.Logger
	channel chan Log
	level LogLevel
}

type Log struct {
	level LogLevel
	message string
	data []zap.Field
}

type LogLevel int8

const(
	Info LogLevel = 0
	Debug LogLevel = 1
	Warning LogLevel = 2
	Error LogLevel = 3
	Fatal LogLevel = 4
)

func newLogChannel(logger *zap.Logger, config *Config) LogChannel  {
	return LogChannel {
		logger:  *logger,
		channel: make(chan Log),
		level: config.Minosse.Log,
	}
}

func (logChannel LogChannel) handleLog()  {
	defer logChannel.logger.Sync()

	for {
		log := <- logChannel.channel

		if log.level >= logChannel.level {
			switch log.level {
			case Info:
				logChannel.logger.Info(log.message, log.data...)
				break
			case Debug:
				logChannel.logger.Debug(log.message, log.data...)
				break
			case Warning:
				logChannel.logger.Warn(log.message, log.data...)
				break
			case Error:
				logChannel.logger.Error(log.message, log.data...)
				break
			case Fatal:
				logChannel.logger.Fatal(log.message, log.data...)
				break
			default:
				logChannel.logger.Error(fmt.Sprintf("MINOSSE: lOG LEVEL %s UNDEFINED", log.level))
				break
			}
		}
	}
}