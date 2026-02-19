// SPDX-License-Identifier: Apache-2.0
package vsock

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Work-Fort/Anvil/pkg/vsock"
	"github.com/spf13/cobra"
)

func newClientCmd() *cobra.Command {
	var (
		vsockPath string
		port      uint32
		timeout   time.Duration
		message   string
	)

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Send a ping to a vsock server",
		Long: `Send a JSON-RPC ping request to a vsock server and wait for a pong response.

This command runs on the host and connects to a Firecracker VM's vsock socket.
It's designed to be non-interactive and suitable for use in automated testing.

The client sends a "ping" JSON-RPC request and expects a matching "pong" response.
If the server doesn't respond within the timeout or returns an error, the command
exits with a non-zero status code.`,
		Example: `  # Ping default vsock socket
  anvil vsock client --vsock-path /tmp/firecracker.vsock

  # Ping with custom port and timeout
  anvil vsock client --vsock-path /tmp/vm.vsock --port 9000 --timeout 5s

  # Ping with custom message
  anvil vsock client --vsock-path /tmp/vm.vsock --message "hello"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create logger that writes to stderr
			logger := log.New(os.Stderr, "[vsock-client] ", log.LstdFlags)

			// Validate vsock-path
			if vsockPath == "" {
				return fmt.Errorf("--vsock-path is required")
			}

			// Create client
			client := vsock.NewClient(vsockPath, port, logger)

			// Send ping with timeout
			logger.Printf("sending ping to %s port %d (timeout: %s)", vsockPath, port, timeout)
			start := time.Now()

			if err := client.PingWithTimeout(message, timeout); err != nil {
				logger.Printf("ping failed after %s: %v", time.Since(start), err)
				return err
			}

			elapsed := time.Since(start)
			logger.Printf("ping successful (round-trip: %s)", elapsed)
			fmt.Printf("PONG received successfully (round-trip: %s)\n", elapsed)

			return nil
		},
	}

	cmd.Flags().StringVar(&vsockPath, "vsock-path", "", "Path to Firecracker vsock Unix socket (required)")
	cmd.Flags().Uint32VarP(&port, "port", "p", 8000, "vsock port to connect to")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Timeout for ping request")
	cmd.Flags().StringVarP(&message, "message", "m", "ping", "Message to send in ping request")

	cmd.MarkFlagRequired("vsock-path")

	return cmd
}
