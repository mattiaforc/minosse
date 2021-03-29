# Minosse

[![Go Report Card](https://goreportcard.com/badge/github.com/mattiaforc/minosse)](https://goreportcard.com/report/github.com/mattiaforc/minosse)

A simple, fast, easily configurable, pocket-sized static file server written in Go.

**Note:** This project is still a work in progress and is nowhere near ready to be used safely in production.

# Features

- Configurable static file server
- Leaky bucket rate limiting
- Blazing fast, coroutine based json logging with [uber/zap](https://github.com/uber-go/zap) logging library
- Dedicated `.toml` config file
- Slim sized (~5.5M) and small (just a few files and ~1k LOC)
- Includes runnable out-of-the-box benchmark load tests with [k6](https://k6.io)! (Docker-ready)
- Currently supports only GET requests and HTTP/1.1

# Configuration

```toml
title = "EXAMPLE MINOSSE CONFIGURATION"

[minosse]
# Log level
# -1 = DISABLED 
# 0 = INFO
# 1 = DEBUG
# 2 = WARNING
# 3 = ERROR
# 4 = FATAL
log = 0
port = 8080
server = "0.0.0.0"
webroot = "public" # This could be a relative path or an absolute one
maxProcessNumbers = 8 # Defaults to GOMAXPROCS

[minosse.connections]
# Leaky bucket rate limiting.
# Maximum number of concurrent connections. 
maxConnections = 500
readTimeout = 30
writeTimeout = 30

[zap]
# Zap logger mode. Refer to https://github.com/uber-go/zap
mode = "development"

[minosse.tls]
X509RootCAPath = "private/rootCA.key"
X509CertPath = "private/server.crt"
X509KeyPath = "private/server.key"
enabled = true
port = 443

```

## TLS configuration

Optional: Add a root CA (Certificate Authority) or create one:
```sh
openssl genrsa -des3 -out rootCA.key 2048
```
and use that private key to generate the root CA certificate:
```sh
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1825 -out rootCA.pem
```

Create the private key for minosse:
```sh
openssl genrsa -out client.key 2048
```
Create the certificate for minosse:
```sh
openssl req -new -key client.key -out client.csr
```

Optional as above: Sign the certificate with the root CA private key:
```sh
openssl x509 -req -in client.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out client.crt -days 825 -sha256
```

# TODOs

- ~~Support HTTPS~~
- Add a CLI interface
- Add authenticated resources (something like nginx.conf)
- Generating custom configuration from CLI command
- Create a complete `Dockerfile` 