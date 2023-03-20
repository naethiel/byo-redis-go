package main

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"os"

	"github.com/inconshreveable/log15"
)

func main() {
	logger := log15.New()
	logger.Info("server started, listening on TCP port 6379")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		logger.Error("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("accepting connection", "error", err)
			os.Exit(1)
		}

		go handleConn(conn, logger)
	}
}

func handleConn(conn net.Conn, log log15.Logger) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Error("reading from conn", "error", err)
				break
			}
		}

		log.Debug("recieving connection", "content", string(buf))

		payload, err := decodeRESP(bufio.NewReader(bytes.NewReader(buf)))
		if err != nil {
			log.Error("decoding RESP object", "error", err)
			break
		}

		log.Debug("decoded RESP object", "object", payload)

		command, args := payload.Children[0], payload.Children[1:]
		switch string(command.Value) {
		case "ping":
			handlePing(conn)
		case "echo":
			handleEcho(conn, args[0].Value)
		default:
			break
		}
	}

	err := conn.Close()

	if err != nil {
		log.Error("closing connection", "error", err)
	}
}

func handlePing(conn net.Conn) error {
	_, err := conn.Write(encodeSimpleString([]byte("PONG")))
	return err
}

func handleEcho(conn net.Conn, arg []byte) error {
	_, err := conn.Write(encodeBulkString(arg))
	return err
}
