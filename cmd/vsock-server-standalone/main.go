// SPDX-License-Identifier: Apache-2.0
// Standalone static vsock server for VM rootfs
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mdlayher/vsock"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

func main() {
	logger := log.New(os.Stderr, "[vsock-server] ", log.LstdFlags)

	listener, err := vsock.Listen(8000, nil)
	if err != nil {
		logger.Fatalf("Failed to create vsock listener: %v", err)
	}
	defer listener.Close()

	logger.Printf("vsock server listening on port 8000")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Println("Received shutdown signal")
		cancel()
		listener.Close()
	}()

	// Accept connections
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Printf("Failed to accept connection: %v", err)
			continue
		}

		logger.Printf("Accepted connection from %s", conn.RemoteAddr())
		go handleConnection(conn, logger)
	}
}

type PingParams struct {
	Message string `json:"message"`
}

func handleConnection(conn net.Conn, logger *log.Logger) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			logger.Printf("Failed to parse request: %v", err)
			continue
		}

		logger.Printf("Received request: method=%s id=%v", req.Method, req.ID)

		var response JSONRPCResponse
		response.JSONRPC = "2.0"
		response.ID = req.ID

		if req.Method == "ping" {
			// Parse ping params to get the message
			var params PingParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				logger.Printf("Failed to parse ping params: %v", err)
				response.Error = map[string]interface{}{
					"code":    -32602,
					"message": "Invalid params",
				}
			} else {
				// Echo back the message
				response.Result = map[string]string{"message": params.Message}
				logger.Printf("Echoing message: %s", params.Message)
			}
		} else {
			response.Error = map[string]interface{}{
				"code":    -32601,
				"message": fmt.Sprintf("Method not found: %s", req.Method),
			}
		}

		if err := encoder.Encode(response); err != nil {
			logger.Printf("Failed to send response: %v", err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Printf("Connection error: %v", err)
	}
}
