package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pelletier/go-toml"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

const SocketReadTimeout = 30
const SocketWriteTimeout = 30

var config Config
var logChannel LogChannel

func main() {
	printMinosse()
	// TODO: Provide defaults
	configure(&config)
	configureLogger()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.Port))
	if err != nil {
		if logChannel.level != DISABLED {
			logChannel.channel <- Log{
				level:   FATAL,
				message: "Error when trying to listen at specified address:port",
				data:    []zap.Field{zap.Error(err)},
			}
		}
	}
	if logChannel.level != DISABLED {
		logChannel.channel <- Log{
			level:   DEBUG,
			message: "Minosse server started",
			data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.Port)},
		}
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
		logChannel.error("WARNING: Could not read minosse configuration file", err)
	}

	err = toml.Unmarshal(confFile, conf)
	if err != nil {
		logChannel.error("WARNING: Error in minosse configuration file", err)
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

	if logChannel.level != DISABLED {
		go logChannel.handleLog()
	}
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

	requestUri = req.URL.String()
	requestMethod = req.Method
	remoteAddr = req.RemoteAddr

	if requestMethod != HTTP_GET_METHOD {
		response := responseMethodNotAllowed()
		_, err = conn.Write(response.toByte())
	}

	var filepath = config.Minosse.WebRoot + filepath.Clean(requestUri)
	var response Response

	stat, err := os.Stat(filepath)
	if err != nil {
		response = responseNotFound()
		statusCode = 404
		_, err = conn.Write(response.toByte())
		if err != nil {
			logChannel.error("Error writing response", err)
		}
		return
	} else {
		response = responseOkNoBody(map[string]string{HEADER_CONTENT_TYPE: mime.TypeByExtension(path.Ext(filepath)), HEADER_CONTENT_LENGTH: strconv.FormatInt(stat.Size(), 10), HEADER_CACHE_CONTROL: HEADER_CACHE_CONTROL_DEFAULT_VALUE, HEADER_CONNECTION: HEADER_CONNECTION_CLOSE, HEADER_LAST_MODIFIED: stat.ModTime().Format(http.TimeFormat), HEADER_DATE: time.Now().Format(http.TimeFormat), HEADER_SERVER: HEADER_SERVER_VALUE})
		statusCode = 200
	}

	_, err = conn.Write(response.responseToByteNoBody())
	if err != nil {
		logChannel.error("Error writing response", err)
		return
	}

	f, err := os.Open(filepath)
	defer f.Close()

	if err != nil {
		logChannel.error("Error opening file", err)
		return
	}

	conn = conn.(*net.TCPConn)
	_, err = io.Copy(conn, f)
	if err != nil {
		logChannel.error("Error writing response", err)
		return
	}
}
