// SPDX-License-Identifier: Apache-2.0
package vsock

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Work-Fort/Anvil/pkg/vsock"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	var port uint32

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start a vsock JSON-RPC server",
		Long: `Start a vsock JSON-RPC server that responds to ping requests with pong responses.

This command runs inside a Firecracker VM and listens on an AF_VSOCK socket.
It's designed to be non-interactive and suitable for use in automated testing.

The server responds to JSON-RPC 2.0 "ping" methods with "pong" responses,
allowing the host to verify that virtio-vsock is working correctly.`,
		Example: `  # Start server on default port 8000
  anvil vsock server

  # Start server on custom port
  anvil vsock server --port 9000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create logger that writes to stderr
			logger := log.New(os.Stderr, "[vsock-server] ", log.LstdFlags)

			// Create server
			server := vsock.NewServer(port, logger)

			// Create context that cancels on SIGINT/SIGTERM
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				logger.Println("received shutdown signal")
				cancel()
			}()

			// Start server (blocks until context cancelled)
			logger.Printf("starting vsock server on port %d", port)
			if err := server.Listen(ctx); err != nil && err != context.Canceled {
				return fmt.Errorf("server error: %w", err)
			}

			logger.Println("server stopped")
			return nil
		},
	}

	cmd.Flags().Uint32VarP(&port, "port", "p", 8000, "vsock port to listen on")

	return cmd
}
