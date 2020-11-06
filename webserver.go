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
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

// SocketReadTimeout constant timeout values for incoming connections
const SocketReadTimeout = 30

// SocketReadTimeout constant timeout values for outgoing connections
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
		level:   Debug,
		message: "Minosse server started",
		data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.Port)},
	}

	newConnections := make(chan net.Conn)
	go func(l net.Listener) {
		for {
			c, err := l.Accept()
			if err != nil {
				logChannel.channel <- Log{
					level:   Error,
					message: "Error accepting new TCP connection",
					data:    []zap.Field{zap.Error(err)},
				}
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
	confFile, err := ioutil.ReadFile("./config/config.example.toml")
	if err != nil {
		logChannel.channel <- Log{
			level:   Error,
			message: "Error reading minosse configuration file",
			data:    []zap.Field{zap.Error(err)},
		}
	}

	err = toml.Unmarshal(confFile, conf)
	if err != nil {
		logChannel.channel <- Log{
			level:   Error,
			message: "Error in minosse configuration file",
			data:    []zap.Field{zap.Error(err)},
		}
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

	// TODO: connection timeout from config
	if err := conn.SetReadDeadline(time.Now().Add(time.Second * SocketReadTimeout)); err != nil {
		logChannel.channel <- Log{
			level:   Error,
			message: "Error setting read deadline",
			data:    []zap.Field{zap.Error(err)},
		}
		return
	}
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second * SocketWriteTimeout)); err != nil {
		logChannel.channel <- Log{
			level:   Error,
			message: "Error setting write deadline",
			data:    []zap.Field{zap.Error(err)},
		}
		return
	}

	buf := bufio.NewReader(conn)

	req, err := http.ReadRequest(buf)
	if err != nil {
		logChannel.channel <- Log{
			level:   Error,
			message: "Error reading request",
			data:    []zap.Field{zap.Error(err)},
		}
		return
	}
	logChannel.channel <- Log{
		level:   Debug,
		message: "Request received",
		data:    []zap.Field{zap.String("URI", req.RequestURI), zap.String("Method", req.Method)},
	}

	content, err := ioutil.ReadFile(filepath.Clean(req.RequestURI[1:]))
	var response Response
	if err != nil {
		response = newResponseBuilder().StatusCode(404).Status("Not Found").Header("Transfer-Encoding", "identity").Header("Content-Type", "text/plain; charset=utf-8").Body([]byte("404 Not Found")).Build()
	} else {
		response = newResponseBuilder().Status("OK").StatusCode(200).Header("Transfer-Encoding", "identity").Header("Content-Type", mime.TypeByExtension(req.RequestURI[strings.IndexRune(req.RequestURI, '.'):])).Header("Content-Length", strconv.Itoa(len(content))).Body(content).Build()
	}

	_, err = conn.Write(response.toByte())
	if err != nil {
		logChannel.channel <- Log{
			level:   Error,
			message: "Error writing response",
			data:    []zap.Field{zap.Error(err)},
		}
		return
	}
}
