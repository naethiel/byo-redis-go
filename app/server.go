package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/inconshreveable/log15"
)

type Service struct {
	Store    Store
	Log      log15.Logger
	Port     int
	Protocol string
}

func (s *Service) Configure() error {
	s.Log = log15.New()
	s.Port = 6379
	s.Store = Store{}
	s.Protocol = "tcp"

	return nil
}

func main() {
	var s Service
	err := s.Configure()
	if err != nil {
		fmt.Printf("Failed to configure service: %s", err.Error())
		os.Exit(1)
		return
	}

	s.Log.Info("server started, listening", "protocol", s.Protocol, "port", s.Port)

	l, err := net.Listen(s.Protocol, fmt.Sprintf("0.0.0.0:%d", s.Port))
	if err != nil {
		s.Log.Error("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			s.Log.Error("accepting connection", "error", err)
			os.Exit(1)
		}

		go s.handleConn(conn)
	}
}

func (s Service) handleConn(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				s.Log.Error("reading from conn", "error", err)
				break
			}
		}

		s.Log.Debug("recieving connection", "content", string(buf))

		payload, err := decodeRESP(bufio.NewReader(bytes.NewReader(buf)))
		if err != nil {
			s.Log.Error("decoding RESP object", "error", err)
			break
		}

		s.Log.Debug("decoded RESP object", "object", payload)

		if len(payload.Children) == 0 {
			s.Log.Error("no command in decoded RESP object", "resp", payload)
			break
		}

		command := payload.Children[0]
		args := make([]RESP, 0, len(payload.Children))
		if len(payload.Children) > 1 {
			args = append(args, payload.Children[1:]...)
		}

		s.Log.Debug("handling command", "type", string(command.Value))

		switch string(command.Value) {
		case "ping":
			err := s.handlePing(conn)
			if err != nil {
				s.Log.Error("handling ping request", "error", err)
			}
		case "echo":
			err := s.handleEcho(conn, args)
			if err != nil {
				s.Log.Error("handling echo request", "error", err)
			}
		case "set":
			err := s.handleSet(conn, args)
			if err != nil {
				s.Log.Error("handling set request", "error", err)
			}
		case "get":
			err := s.handleGet(conn, args)
			if err != nil {
				s.Log.Error("handling get request", "error", err)
			}
		default:
			break
		}
	}

	err := conn.Close()

	if err != nil {
		s.Log.Error("closing connection", "error", err)
	}
}

func (s Service) handleGet(conn net.Conn, args []RESP) error {
	if len(args) == 0 {
		return fmt.Errorf("not enough args provided to GET handler")
	}

	val, exists := s.Store.Get(args[0].Value)

	if !exists {
		s.Log.Debug("value not found in store", "key", string(args[0].Value))
	}

	_, err := conn.Write(encodeBulkString(val))
	return err
}

func (s Service) handleSet(conn net.Conn, args []RESP) error {
	if len(args) < 2 {
		return fmt.Errorf("not enough arguments provided to SET handler")
	}

	s.Store.Set(args[0].Value, args[1].Value)

	_, err := conn.Write(encodeSimpleString([]byte("OK")))
	return err
}

func (s *Service) handlePing(conn net.Conn) error {
	_, err := conn.Write(encodeSimpleString([]byte("PONG")))
	return err
}

func (s *Service) handleEcho(conn net.Conn, args []RESP) error {
	var arg []byte
	if len(args) > 0 {
		arg = args[0].Value
	}

	_, err := conn.Write(encodeBulkString(arg))
	return err
}
