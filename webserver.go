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
)

const SocketReadTimeout = 30
const SocketWriteTimeout = 30

var config Config

func main() {
	printMinosse()
	configure(&config)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Minosse.Server, config.Minosse.Port))
	if err != nil {
		log.Fatalf("Errore: %v", err)
	}
	fmt.Printf("Server listening on port %d\n", config.Minosse.Port)

	newConnections := make(chan net.Conn)
	go func(l net.Listener) {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Printf("Error accepting new TCP connection: %s", err.Error())
				newConnections <- nil
				return
			}
			newConnections <- c
		}
	}(listener)

	for {
		select {
		case c := <-newConnections:
			go handleConnection(c)
		}
	}
}

func configure(conf *Config) {
	confFile, err := ioutil.ReadFile("config.example.toml")
	if err != nil {
		log.Printf("Error reading minosse configuration file")
	}

	err = toml.Unmarshal(confFile, conf)
	if err != nil {
		log.Printf("Error in minosse configuration file. Error: %s", err.Error())
	}
}

func printMinosse() {
	asciiArt :=
		`
 __   __  ___   __    _  _______  _______  _______  _______ 
|  |_|  ||   | |  |  | ||       ||       ||       ||       |
|       ||   | |   |_| ||   _   ||  _____||  _____||    ___|
|       ||   | |       ||  | |  || |_____ | |_____ |   |___ 
|       ||   | |  _    ||  |_|  ||_____  ||_____  ||    ___|
| ||_|| ||   | | | |   ||       | _____| | _____| ||   |___ 
|_|   |_||___| |_|  |__||_______||_______||_______||_______|`
	fmt.Println(asciiArt)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// TODO: Defer event sync
	//defer logger.Sync()

	// TODO: connection timeout from config
	if err := conn.SetReadDeadline(time.Now().Add(time.Second * SocketReadTimeout)); err != nil {
		log.Printf("Error setting read deadline: %s", err.Error())
		return
	}
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second * SocketWriteTimeout)); err != nil {
		log.Printf("Error setting write deadline: %s", err.Error())
		return
	}

	buf := bufio.NewReader(conn)

	req, err := http.ReadRequest(buf)
	if err != nil {
		log.Printf("Error reading request: %s", err.Error())
		return
	}
	// log.Printf("Request: %v", req)
	// TODO: Send log data to channel
	// logger.Info("Request received", zap.String("URI", req.RequestURI), zap.String("Method", req.Method))
	// log.Printf("Requested file: %s", req.RequestURI)

	content, err := ioutil.ReadFile(filepath.Clean(req.RequestURI[1:]))
	var response Response

	if err != nil {
		response = newResponseBuilder().StatusCode(404).Status("Not Found").Header("Transfer-Encoding", "identity").Header("Content-Type", "text/plain; charset=utf-8").Body([]byte("404 Not Found")).Build()
	} else {
		response = newResponseBuilder().Status("OK").StatusCode(200).Header("Transfer-Encoding", "identity").Header("Content-Type", mime.TypeByExtension(req.RequestURI[strings.IndexRune(req.RequestURI, '.'):])).Header("Content-Length", strconv.Itoa(len(content))).Body(content).Build()
	}

	_, err = conn.Write(response.toByte())
	if err != nil {
		log.Printf("Error writing response: %s", err.Error())
		return
	}
}
