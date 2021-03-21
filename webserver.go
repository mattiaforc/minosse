package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
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

	"github.com/panjf2000/ants/v2"
	"github.com/pelletier/go-toml"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

const SocketReadTimeout = 30
const SocketWriteTimeout = 30

var config Config
var logChannel LogChannel

func main() {
	defer ants.Release()

	PrintMinosse()
	// TODO: Provide defaults
	configure(&config)
	configureLogger()

	newConnections := make(chan net.Conn)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.Port))
	if err != nil {
		logChannel.fatalError("Error when trying to listen at specified address:port", err)
		return
	}
	if logChannel.level != DISABLED {
		logChannel.channel <- Log{
			level:   DEBUG,
			message: "Minosse server started",
			data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.Port), zap.String("root", config.Minosse.WebRoot)},
		}
	}
	go listen(listener, newConnections)

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

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: rootCAPool}

		tlsListener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.TLS.Port), tlsConfig)
		if err != nil {
			logChannel.fatalError("Error when trying to listen at specified address:port", err)
			return
		}
		if logChannel.level != DISABLED {
			logChannel.channel <- Log{
				level:   DEBUG,
				message: "Serving with TLS enabled on: ",
				data:    []zap.Field{zap.String("address", config.Minosse.Server), zap.Int("port", config.Minosse.TLS.Port)},
			}
		}

		go listen(tlsListener, newConnections)
	}

	var rl ratelimit.Limiter
	if config.Minosse.Connections.MaxConnections > 0 {
		rl = ratelimit.New(config.Minosse.Connections.MaxConnections)
	} else {
		rl = ratelimit.NewUnlimited()
	}

	var pool *ants.PoolWithFunc

	if config.Ants.Enabled {
		pool, err = ants.NewPoolWithFunc(config.Ants.PoolSize, func(c interface{}) {
			handleConnection(c.(net.Conn))
		}, func(opts *ants.Options) {
			options := ants.Options{
				ExpiryDuration:   config.Ants.ExpiryDuration,
				PreAlloc:         config.Ants.PreAlloc,
				MaxBlockingTasks: config.Ants.MaxBlockingTasks,
				Nonblocking:      config.Ants.Nonblocking,
			}
			*opts = options
		})
	}

	if config.Ants.Enabled {
		for {
			select {
			case c := <-newConnections:
				rl.Take()
				err = pool.Invoke(c)
				if err != nil {
					logChannel.error("Pool error", err)
				}
			}
		}
	} else {
		for {
			select {
			case c := <-newConnections:
				rl.Take()
				go handleConnection(c)
			}
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

func listen(l net.Listener, newConnections chan net.Conn) {
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(config.Minosse.Connections.ReadTimeout))); err != nil {
		logChannel.error("Error setting read deadline", err)
		return
	}
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(config.Minosse.Connections.WriteTimeout))); err != nil {
		logChannel.error("Error setting write deadline", err)
		return
	}

	var req *http.Request
	var response Response

	req, err := http.ReadRequest(bufio.NewReader(conn))
	defer logChannel.logWholeRequest(req, &response)

	if err != nil {
		logChannel.error("Error reading request", err)
		return
	}

	if req.Method != HTTP_GET_METHOD {
		response := ResponseMethodNotAllowed()
		_, err = conn.Write(response.ToByte())
		return
	}

	var pathFile = config.Minosse.WebRoot + filepath.Clean(req.URL.String())

	f, err := os.Open(pathFile)
	if err != nil {
		logChannel.error("File not found", err)
		response = ResponseNotFound()
		_, err = conn.Write(response.ToByte())
		if err != nil {
			logChannel.error("Error writing response", err)
		}
		return
	}
	stat, err := f.Stat()
	if err != nil {
		logChannel.error("Error during file stat", err)
		response = ResponseInternalServerError()
		_, err = conn.Write(response.ToByte())
		if err != nil {
			logChannel.error("Error writing response", err)
		}
		return
	} else {
		response = ResponseOkNoBody(map[string]string{HEADER_CONTENT_TYPE: mime.TypeByExtension(path.Ext(pathFile)), HEADER_CONTENT_LENGTH: strconv.FormatInt(stat.Size(), 10), HEADER_CACHE_CONTROL: HEADER_CACHE_CONTROL_DEFAULT_VALUE, HEADER_CONNECTION: HEADER_CONNECTION_CLOSE, HEADER_LAST_MODIFIED: stat.ModTime().Format(http.TimeFormat), HEADER_DATE: time.Now().Format(http.TimeFormat), HEADER_SERVER: HEADER_SERVER_VALUE})
	}

	_, err = conn.Write(response.ResponseToByteNoBody())
	if err != nil && err != io.EOF {
		logChannel.error("Error writing response", err)
		return
	}

	_, err = io.Copy(conn, f)
	if err != nil {
		logChannel.error("Error writing response", err)
		return
	}
}
