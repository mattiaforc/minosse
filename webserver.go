package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

const OSX_OS = "darwin/amd64"
const GENERIC_ERROR_MESSAGE_LOG string = "Error reading request"
const TCP_CONNECTION_ERROR_MESSAGE_LOG string = "Error accepting new TCP connection"
const SocketReadTimeout = 30
const SocketWriteTimeout = 30

var config Config
var logChannel LogChannel

func main() {
	printMinosse()
	configure(&config)
	configureLogger()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.Port))
	if err != nil {
		log.Fatalf("Errore: %v", err)
	}
	logChannel.channel <- Log{
		level:   DEBUG,
		message: "Minosse server started",
		data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.Port)},
	}

	newConnections := make(chan net.Conn)
	go func(l net.Listener) {
		for {
			c, err := l.Accept()
			if err != nil {
				logChannel.error(TCP_CONNECTION_ERROR_MESSAGE_LOG, err)
				newConnections <- nil
				return
			}
			newConnections <- c
		}
	}(listener)

	var rl ratelimit.Limiter
	if config.Minosse.Connections.MaxConnections > 0 {
		rl = ratelimit.New(config.Minosse.Connections.MaxConnections)
	} else {
		rl = ratelimit.NewUnlimited()
	}

	for {
		select {
		case c := <-newConnections:
			rl.Take()
			go handleConnection(c)
		}
	}
}

func configure(conf *Config) {
	// TODO: read from cli --flags
	confFile, err := ioutil.ReadFile("./config/config.example.toml")
	if err != nil {
		logChannel.error("Error reading minosse configuration file", err)
	}

	err = toml.Unmarshal(confFile, conf)
	if err != nil {
		logChannel.error("Error in minosse configuration file", err)
	}
}

func configureLogger() {
	var logger *zap.Logger
	switch config.Zap.Mode {
	// TODO: Handle errors
	case "development":
		logger, _ = zap.NewDevelopment()
		break
	case "production":
		logger, _ = zap.NewProduction()
		break
	default:
		logger = zap.NewExample()
	}
	logChannel = newLogChannel(logger, &config)
	go logChannel.handleLog()
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	var requestUri string
	var requestMethod string
	var statusCode int
	var remoteAddr string
	defer logChannel.logRequest(time.Now(), &requestUri, &requestMethod, &statusCode, &remoteAddr)

	if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(config.Minosse.Connections.ReadTimeout))); err != nil {
		logChannel.error("Error setting read deadline", err)
		return
	}
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(config.Minosse.Connections.WriteTimeout))); err != nil {
		logChannel.error("Error setting write deadline", err)
		return
	}

	buf := bufio.NewReader(conn)
	req, err := http.ReadRequest(buf)
	if err != nil {
		logChannel.error("Error reading request", err)
		return
	}

	requestUri = req.RequestURI[1:]
	requestMethod = req.Method
	remoteAddr = req.RemoteAddr

	if requestMethod != HTTP_GET_METHOD {
		response := responseMethodNotAllowed()
		_, err = conn.Write(response.toByte())
	}

	content, err := ioutil.ReadFile(filepath.Clean(requestUri))
	var response Response

	if err != nil {
		response = responseNotFound()
		statusCode = 404
	} else {
		response = responseOk(content, map[string]string{"Content-Type": mime.TypeByExtension(req.RequestURI[strings.IndexRune(req.RequestURI, '.'):]), "Content-Length": strconv.Itoa(len(content)), "Cache-Control": "public, max-age=604800, immutable"})
		if runtime.GOOS != OSX_OS && !strings.Contains(runtime.Version(), "1.15") {
			response.Header("Content-Encoding", "identity")
		}
		statusCode = 200
	}

	_, err = conn.Write(response.toByte())
	if err != nil {
		logChannel.error("Error writing response", err)
		return
	}
}
