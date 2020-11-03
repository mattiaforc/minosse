package main

type Config struct {
	Minosse Minosse
	Zap Zap
}

type Minosse struct {
	Server string
	Port int32
	Log string
	Connections Connections
}

type Connections struct {
	ReadTimeout int
	WriteTimeout int
	MaxConnections int32
}

type Zap struct {
	Mode string
}