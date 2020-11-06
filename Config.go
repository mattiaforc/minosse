package main

type Config struct {
	Minosse Minosse
	Zap Zap
}

type Minosse struct {
	Server string
	Port int
	Log LogLevel
	Connections Connections
}

type Connections struct {
	ReadTimeout int
	WriteTimeout int
	MaxConnections int
}

type Zap struct {
	Mode string
}