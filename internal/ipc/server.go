package ipc

import (
	"encoding/json"
	"log"
	"net"
)

// Handler processes a command and returns a response.
type Handler func(cmd Command) Response

// Serve listens on the Unix socket and dispatches commands to handler.
func Serve(socketPath string, handler Handler) error {
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleConn(conn, handler)
	}
}

func handleConn(conn net.Conn, handler Handler) {
	defer conn.Close()

	var cmd Command
	if err := json.NewDecoder(conn).Decode(&cmd); err != nil {
		log.Printf("ipc: decode error: %v", err)
		return
	}

	resp := handler(cmd)
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		log.Printf("ipc: encode error: %v", err)
	}
}
