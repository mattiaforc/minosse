package main

// Config Central configuration structure (minosse, zap logger, etc...)
type Config struct {
	Minosse Minosse
	Zap     Zap
}

// Minosse Main minosse configuration structure
type Minosse struct {
	Server           string
	Port             int
	WebRoot          string
	Log              LogLevel
	Connections      Connections
	TLS              TLS
	Gzip             GZip
	MaxProcessNumber int
}

// GZip configurations
type GZip struct {
	Enabled   bool
	Level     int
	Threshold int64
	Exclude   string
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

type TLS struct {
	Enabled        bool
	Port           int
	X509CertPath   string
	X509KeyPath    string
	X509RootCAPath string
}
