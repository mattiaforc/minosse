package main

// Config Central configuration structure (minosse, zap logger, etc...)
type Config struct {
	Minosse Minosse
	Zap     Zap
}

// Minosse Main minosse configuration structure
type Minosse struct {
	Server      string
	Port        int
	WebRoot     string
	Log         LogLevel
	Connections Connections
}

// Connections Configurations regarding connections
type Connections struct {
	ReadTimeout    int
	WriteTimeout   int
	MaxConnections int
}

// Zap Configuration for zap logger
type Zap struct {
	Mode string
}
