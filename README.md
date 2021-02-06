# Minosse

[![Go Report Card](https://goreportcard.com/badge/github.com/mattiaforc/minosse)](https://goreportcard.com/report/github.com/mattiaforc/minosse)

A simple, fast, easily configurable, pocket-sized static file server written in Go.

**Note:** This project is still a work in progress and is nowhere near ready to be used safely in production.

# Features

- Configurable static file server
- Concurrent requests handling
- Leaky bucket rate limiting
- Blazing fast, coroutine based json logging with [uber/zap](https://github.com/uber-go/zap) logging library
- Easily configurable with dedicated `.toml` config file
- Slim sized (~5.5M) and small (a few files and ~300 LOC)
- Includes runnable out-of-the-box benchmark load tests with [k6](https://k6.io)! (requires Docker)
- Currently supports only GET requests and HTTP/1.1

# Configuration

```toml
title = "Example minosse configuration"

[minosse]
# Log level
# 0 = INFO
# 1 = DEBUG
# 2 = WARNING
# 3 = ERROR
# 4 = FATAL
log = 0
port = 8080
server = "0.0.0.0"

[minosse.connections]
# Leaky bucket rate limiting.
# Maximum number of concurrent connections. 
maxConnections = 500
readTimeout = 30
writeTimeout = 30

[zap]
# Zap logger mode. Refer to https://github.com/uber-go/zap
mode = "development"

```

# TODOs

- Add authenticated resources (something like .htaccess)
- Add a CLI interface
- Create a complete `Dockerfile` 