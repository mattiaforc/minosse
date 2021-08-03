package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/valyala/fasthttp"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const SocketReadTimeout = 30
const SocketWriteTimeout = 30

var config Config
var logChannel LogChannel
var excludePattern *regexp.Regexp

func main() {

	PrintMinosse()
	configure(&config)
	configureLogger()
	applyDefaultConfigValues(&config)

	newConnections := make(chan net.Conn)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.Port))
	if err != nil {
		logChannel.fatalError("Error when trying to listen at specified address:port", err)
		return
	}
	logChannel.channel <- Log{
		level:   INFO,
		message: "Minosse server started",
		data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.Port), zap.String("root", config.Minosse.WebRoot)},
	}

	if config.Minosse.TLS.Enabled {
		cert, err := tls.LoadX509KeyPair(config.Minosse.TLS.X509CertPath, config.Minosse.TLS.X509KeyPath)
		if err != nil {
			logChannel.fatalError("Fatal error while loading x509 keypair", err)
			return
		}
		var rootCAPool *x509.CertPool
		if config.Minosse.TLS.X509RootCAPath != "" {
			rootCAPool = x509.NewCertPool()
			rootCA, err := ioutil.ReadFile(config.Minosse.TLS.X509RootCAPath)
			if err != nil {
				logChannel.error("Could not read root CA file", err)
				return
			}
			ok := rootCAPool.AppendCertsFromPEM(rootCA)
			if !ok {
				logChannel.error("Unable to use supplied root CA cert. Make sure it is a valid certificate", nil)
				return
			}
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: rootCAPool, MinVersion: tls.VersionTLS12}

		tlsListener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.TLS.Port), tlsConfig)
		if err != nil {
			logChannel.fatalError("Error when trying to listen at specified address:port", err)
			return
		}
		logChannel.channel <- Log{
			level:   INFO,
			message: "Serving with TLS enabled on: ",
			data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.TLS.Port)},
		}

		go listen(tlsListener, newConnections)
	}

	var rl ratelimit.Limiter
	if config.Minosse.Connections.MaxConnections > 0 {
		rl = ratelimit.New(config.Minosse.Connections.MaxConnections)
	} else {
		rl = ratelimit.NewUnlimited()
	}

	maxWorkers := config.Minosse.MaxProcessNumber
	for w := 0; w < maxWorkers; w++ {
		go worker(newConnections, rl)
	}

	listen(listener, newConnections)
}

func configure(conf *Config) {
	// TODO: read from cli --flags
	confFile, err := ioutil.ReadFile("./config/config.example.toml")
	if err != nil {
		logChannel.error("WARNING: Could not read minosse configuration file", err)
	}

	err = toml.Unmarshal(confFile, conf)
	if err != nil {
		logChannel.error("WARNING: Error in minosse configuration file", err)
	}
}

func applyDefaultConfigValues(conf *Config) {
	// Server
	if conf.Minosse.Server == "" {
		conf.Minosse.Server = "localhost"
		logChannel.channel <- Log{level: INFO, message: "Using localhost as server url"}
	}
	// Server port
	if conf.Minosse.Port == 0 {
		logChannel.channel <- Log{level: INFO, message: "Using 8080 as default server port"}
		conf.Minosse.Port = 8080
	}
	// Max process number
	if conf.Minosse.MaxProcessNumber == 0 {
		conf.Minosse.MaxProcessNumber = runtime.GOMAXPROCS(0)
		logChannel.channel <- Log{level: INFO, message: "Using default GOMAXPROCS workers", data: []zap.Field{zap.Int("GOMAXPROCS", runtime.GOMAXPROCS(0))}}
	} else {
		logChannel.channel <- Log{level: INFO, message: "Using specified amount of workers", data: []zap.Field{zap.Int("Workers", conf.Minosse.MaxProcessNumber)}}
	}
	// X509 - TLS
	if conf.Minosse.TLS.Enabled {
		if conf.Minosse.TLS.X509CertPath == "" {
			logChannel.fatalError("TLS is enabled, but no X509 certificate path was specified in current configuration", nil)
		}
		if conf.Minosse.TLS.X509RootCAPath == "" {
			logChannel.fatalError("TLS is enabled, but no X509 Root CA path was specified in current configuration", nil)
		}
		if conf.Minosse.TLS.X509KeyPath == "" {
			logChannel.fatalError("TLS is enabled, but no X509 key path was specified in current configuration", nil)
		}
		if conf.Minosse.TLS.Port == 0 {
			conf.Minosse.TLS.Port = 8000
		}
	}
	// Web root
	if conf.Minosse.WebRoot == "" {
		logChannel.fatalError("No webroot was specified in current configuration", nil)
	} else {
		logChannel.channel <- Log{level: INFO, message: "Serving static files", data: []zap.Field{zap.String("Directory", conf.Minosse.WebRoot)}}
	}
	// Connection timeout
	if conf.Minosse.Connections.ReadTimeout == 0 {
		logChannel.channel <- Log{level: INFO, message: "Using default connection read timeout of 30 seconds"}
		conf.Minosse.Connections.ReadTimeout = SocketReadTimeout
	}
	if conf.Minosse.Connections.WriteTimeout == 0 {
		logChannel.channel <- Log{level: INFO, message: "Using default connection write timeout of 30 seconds"}
		conf.Minosse.Connections.WriteTimeout = SocketWriteTimeout
	}
	// Gzip
	if conf.Minosse.Gzip.Enabled {
		if conf.Minosse.Gzip.Exclude == "" {
			logChannel.channel <- Log{level: INFO, message: "Using default gzip configuration will NOT compress images and pdf files"}
			conf.Minosse.Gzip.Exclude = "(jpeg|jpg|png|pdf)$"
		}
		if conf.Minosse.Gzip.Level == 0 {
			logChannel.channel <- Log{level: INFO, message: "Using default gzip compression level. You can specify the compression level in the configuration file; possible values range from 1 (Best speed) to 9 (Best compression)"}
			conf.Minosse.Gzip.Level = gzip.DefaultCompression
		} else if conf.Minosse.Gzip.Level > 9 {
			logChannel.fatalError("The specified gzip level is not valid. Possible values range from 1 (Best speed) to 9 (Best compression)", nil)
		}
		if conf.Minosse.Gzip.Threshold == 0 {
			logChannel.channel <- Log{level: INFO, message: "Using default gzip file-size threshold. File under 1.5KB will not be compressed."}
			conf.Minosse.Gzip.Threshold = 1500
		} else if conf.Minosse.Gzip.Threshold < 0 {
			logChannel.fatalError("The specified gzip file size threshold is invalid because it is negative.", nil)
		}
		excludePattern = regexp.MustCompile(conf.Minosse.Gzip.Exclude)
	}
}

func configureLogger() {
	var logger *zap.Logger
	var conf zap.Config

	switch config.Zap.Mode {
	case "development":
		conf = zap.NewDevelopmentConfig()

	case "production":
		conf = zap.NewProductionConfig()

	default:
		log.Fatalf("Zap logger mode %s is invalid. Possible values are: development | production", config.Zap.Mode)
	}

	conf.Level.SetLevel(zapcore.Level(config.Minosse.Log - 1))
	logger, err := conf.Build()
	if err != nil {
		log.Fatalf("Could not instantiate zap logger. %v", err)
	}
	logChannel = newLogChannel(logger, &config)

	if logChannel.level != DISABLED {
		go logChannel.handleLog()
	} else {
		go func(channel LogChannel) {
			for {
				<-channel.channel
			}
		}(logChannel)
	}
}

func listen(l net.Listener, newConnections chan net.Conn) {
	defer close(newConnections)
	defer close(logChannel.channel)

	for {
		c, err := l.Accept()
		if err != nil {
			logChannel.error(CONNECTION_ERROR_MESSAGE_LOG, err)
			newConnections <- nil
			return
		}
		newConnections <- c
	}
}

func worker(newConnections chan net.Conn, rl ratelimit.Limiter) {
	for c := range newConnections {
		rl.Take()
		err := fasthttp.ServeConn(c, handleConnection)
		if err != nil {
			logChannel.error("Error while serving connection", err)
		}
	}
}

func gzipFilter(f os.FileInfo) bool {
	return f.Size() > config.Minosse.Gzip.Threshold && !excludePattern.MatchString(f.Name())
}

func handleConnection(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	var gzb bytes.Buffer
	defer logChannel.logWholeRequest(ctx, &start)

	if !bytes.Equal(ctx.Method(), HTTP_GET_METHOD) {
		// TODO: refactor
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		ctx.SetBodyString(HTTP_NOT_ALLOWED)
		ctx.Response.Header.Set(HEADER_CONTENT_TYPE, "text/plain; charset=utf-8")
		return
	}

	gzipEnabled := false
	if ctx.Request.Header.HasAcceptEncoding(GZIP) {
		gzipEnabled = config.Minosse.Gzip.Enabled
	}

	pathFile := config.Minosse.WebRoot + string(ctx.Path())
	f, err := os.Open(pathFile)
	if err != nil {
		logChannel.error("File not found", err)
		// TODO: refactor
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(HTTP_NOT_FOUND_BODY)
		ctx.Response.Header.Set(HEADER_CONTENT_TYPE, "text/plain; charset=utf-8")
		return
	}
	stat, err := f.Stat()
	if err != nil {
		logChannel.error("Error during file stat", err)
		// TODO: refactor
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(HTTP_INTERNAL_SERVER_ERROR)
		ctx.Response.Header.Set(HEADER_CONTENT_TYPE, "text/plain; charset=utf-8")
		return
	} else {
		var encoding string
		var contentLength string
		gzipEnabled = gzipEnabled && gzipFilter(stat)
		if gzipEnabled {
			encoding = "gzip"
			gzipWriter, _ := gzip.NewWriterLevel(&gzb, config.Minosse.Gzip.Level)
			if _, err := io.Copy(gzipWriter, f); err != nil {
				logChannel.error("Error during gzip compression", err)
				return
			}
			if err := gzipWriter.Close(); err != nil {
				logChannel.error("Error while closing gzip compression", err)
				return
			}
			contentLength = strconv.FormatInt(int64(gzb.Len()), 10)
		} else {
			encoding = "identity"
			contentLength = strconv.FormatInt(stat.Size(), 10)
		}
		// TODO: refactor
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Response.Header.Set(HEADER_CONTENT_TYPE, mime.TypeByExtension(path.Ext(pathFile)))
		ctx.Response.Header.Set(HEADER_CONTENT_LENGTH, contentLength)
		ctx.Response.Header.Set(HEADER_CONTENT_ENCODING, encoding)
		ctx.Response.Header.Set(HEADER_CACHE_CONTROL, HEADER_CACHE_CONTROL_DEFAULT_VALUE)
		ctx.Response.Header.Set(HEADER_CONNECTION, HEADER_CONNECTION_CLOSE)
		ctx.Response.Header.Set(HEADER_LAST_MODIFIED, stat.ModTime().Format(http.TimeFormat))
		ctx.Response.Header.Set(HEADER_DATE, time.Now().Format(http.TimeFormat))
		ctx.Response.Header.Set(HEADER_SERVER, HEADER_SERVER_VALUE)
	}

	if gzipEnabled {
		if _, err := io.Copy(ctx.Response.BodyWriter(), &gzb); err != nil {
			logChannel.error("Error writing response", err)
			return
		}
	} else {
		if _, err := io.Copy(ctx.Response.BodyWriter(), f); err != nil {
			logChannel.error("Error writing response", err)
			return
		}
	}
}
