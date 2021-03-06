package main

import "time"

// Config Central configuration structure (minosse, zap logger, etc...)
type Config struct {
	Minosse Minosse
	Ants    Ants
	Zap     Zap
}

// Minosse Main minosse configuration structure
type Minosse struct {
	Server      string
	Port        int
	WebRoot     string
	Log         LogLevel
	Connections Connections
	TLS         TLS
}

// Connections Configurations regarding connections
type Connections struct {
	ReadTimeout    int
	WriteTimeout   int
	MaxConnections int
}

// Ants Configurations regarding Ants coroutine pool
type Ants struct {
	Enabled          bool
	PoolSize         int
	ExpiryDuration   time.Duration
	PreAlloc         bool
	MaxBlockingTasks int
	Nonblocking      bool
}

// Zap Configuration for zap logger
type Zap struct {
	Mode string
}

type TLS struct {
	Enabled        bool
	Port           int
	X509CertPath   string
	X509KeyPath    string
	X509RootCAPath string
}
