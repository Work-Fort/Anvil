// SPDX-License-Identifier: Apache-2.0
package vsock

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/mdlayher/vsock"
)

// Server represents a vsock JSON-RPC server
type Server struct {
	port   uint32
	logger *log.Logger
}

// NewServer creates a new vsock server
func NewServer(port uint32, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Server{
		port:   port,
		logger: logger,
	}
}

// Listen starts the vsock server and listens for connections
func (s *Server) Listen(ctx context.Context) error {
	// Create vsock listener on the specified port
	listener, err := vsock.Listen(s.port, nil)
	if err != nil {
		return fmt.Errorf("failed to create vsock listener on port %d: %w", s.port, err)
	}
	defer listener.Close()

	s.logger.Printf("vsock server listening on port %d", s.port)

	// Accept connections in a loop
	for {
		select {
		case <-ctx.Done():
			s.logger.Println("server shutting down")
			return ctx.Err()
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			s.logger.Printf("failed to accept connection: %v", err)
			continue
		}

		s.logger.Printf("accepted connection from %s", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single vsock connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Read JSON-RPC request (newline-delimited)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				s.logger.Printf("error reading from connection: %v", err)
			}
			return
		}

		s.logger.Printf("received: %s", string(line))

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.logger.Printf("failed to parse JSON-RPC request: %v", err)
			resp := NewErrorResponse(nil, ErrCodeParseError, "Parse error")
			s.sendResponse(writer, resp)
			continue
		}

		// Handle the request
		resp := s.handleRequest(&req)
		s.sendResponse(writer, resp)
	}
}

// handleRequest processes a JSON-RPC request and returns a response
func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case MethodPing:
		return s.handlePing(req)
	default:
		return NewErrorResponse(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

// handlePing handles a ping request and returns a pong response
func (s *Server) handlePing(req *JSONRPCRequest) *JSONRPCResponse {
	// Parse ping parameters
	var params PingParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	s.logger.Printf("ping received: %s", params.Message)

	// Create pong response
	resp, err := NewPongResponse(req.ID, params.Message)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create pong response")
	}

	return resp
}

// sendResponse sends a JSON-RPC response over the connection
func (s *Server) sendResponse(writer *bufio.Writer, resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Printf("failed to marshal response: %v", err)
		return
	}

	s.logger.Printf("sending: %s", string(data))

	// Write response with newline delimiter
	if _, err := writer.Write(append(data, '\n')); err != nil {
		s.logger.Printf("failed to write response: %v", err)
		return
	}

	if err := writer.Flush(); err != nil {
		s.logger.Printf("failed to flush response: %v", err)
		return
	}
}
