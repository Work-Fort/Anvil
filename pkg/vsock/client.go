// SPDX-License-Identifier: Apache-2.0
package vsock

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	fcvsock "github.com/firecracker-microvm/firecracker-go-sdk/vsock"
)

// Client represents a vsock JSON-RPC client
type Client struct {
	vsockPath string
	port      uint32
	logger    *log.Logger
	conn      io.ReadWriteCloser
	reader    *bufio.Reader
}

// NewClient creates a new vsock client
func NewClient(vsockPath string, port uint32, logger *log.Logger) *Client {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Client{
		vsockPath: vsockPath,
		port:      port,
		logger:    logger,
	}
}

// Connect establishes a persistent connection to the vsock server
func (c *Client) Connect(ctx context.Context) error {
	if c.conn != nil {
		return fmt.Errorf("already connected")
	}

	c.logger.Printf("connecting to %s port %d", c.vsockPath, c.port)
	conn, err := fcvsock.DialContext(ctx, c.vsockPath, c.port)
	if err != nil {
		return fmt.Errorf("failed to connect to vsock: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.logger.Printf("connected to vsock server")
	return nil
}

// Close closes the persistent connection
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	return err
}

// Ping sends a ping request and waits for a pong response
// If a persistent connection exists, it reuses it; otherwise creates a temporary connection
func (c *Client) Ping(ctx context.Context, message string) error {
	// Use persistent connection if available, otherwise create temporary one
	var conn io.ReadWriteCloser
	var reader *bufio.Reader
	var shouldClose bool

	if c.conn != nil {
		// Reuse existing connection
		conn = c.conn
		reader = c.reader
		shouldClose = false
	} else {
		// Create temporary connection
		c.logger.Printf("connecting to %s port %d", c.vsockPath, c.port)
		tempConn, err := fcvsock.DialContext(ctx, c.vsockPath, c.port)
		if err != nil {
			return fmt.Errorf("failed to connect to vsock: %w", err)
		}
		conn = tempConn
		reader = bufio.NewReader(conn)
		shouldClose = true
		defer func() {
			if shouldClose {
				conn.Close()
			}
		}()
		c.logger.Printf("connected to vsock server")
	}

	// Create ping request
	req, err := NewPingRequest(1, message)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	// Marshal and send request
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.Printf("sending: %s", string(reqData))

	// Write request with newline delimiter
	if _, err := conn.Write(append(reqData, '\n')); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	// Read response with timeout
	// Create a channel to receive the response
	respChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			errChan <- err
			return
		}
		respChan <- line
	}()

	// Wait for response or timeout
	select {
	case <-ctx.Done():
		return fmt.Errorf("request cancelled: %w", ctx.Err())
	case err := <-errChan:
		return fmt.Errorf("failed to read response: %w", err)
	case respData := <-respChan:
		c.logger.Printf("received: %s", string(respData))

		// Parse response
		var resp JSONRPCResponse
		if err := json.Unmarshal(respData, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Check for error response
		if resp.Error != nil {
			return fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		// Parse pong result
		var result PongResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return fmt.Errorf("failed to parse pong result: %w", err)
		}

		c.logger.Printf("pong received: %s", result.Message)

		// Verify the message matches
		if result.Message != message {
			return fmt.Errorf("pong message mismatch: expected %q, got %q", message, result.Message)
		}

		return nil
	}
}

// PingWithTimeout sends a ping with a specified timeout
func (c *Client) PingWithTimeout(message string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.Ping(ctx, message)
}
