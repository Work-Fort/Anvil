// SPDX-License-Identifier: Apache-2.0
package vsock

import (
	"encoding/json"
	"fmt"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Method names
const (
	MethodPing = "ping"
	MethodPong = "pong"
)

// PingParams represents parameters for a ping request
type PingParams struct {
	Message string `json:"message"`
}

// PongResult represents the result of a pong response
type PongResult struct {
	Message string `json:"message"`
}

// NewPingRequest creates a new JSON-RPC ping request
func NewPingRequest(id interface{}, message string) (*JSONRPCRequest, error) {
	params, err := json.Marshal(PingParams{Message: message})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ping params: %w", err)
	}

	return &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  MethodPing,
		Params:  params,
		ID:      id,
	}, nil
}

// NewPongResponse creates a new JSON-RPC pong response
func NewPongResponse(id interface{}, message string) (*JSONRPCResponse, error) {
	result, err := json.Marshal(PongResult{Message: message})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pong result: %w", err)
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}, nil
}

// NewErrorResponse creates a new JSON-RPC error response
func NewErrorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

// Error codes
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)
