package ai

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const defaultPort = 9711

// Bridge manages the Python AI brain subprocess.
type Bridge struct {
	cmd   *exec.Cmd
	addr  string
	port  int
	ready bool
	mu    sync.Mutex
}

// NewBridge creates a new Bridge that manages the Python brain process.
// If port is 0, the default port 9711 is used.
func NewBridge(port int) *Bridge {
	if port == 0 {
		port = defaultPort
	}
	return &Bridge{
		port: port,
		addr: fmt.Sprintf("http://127.0.0.1:%d", port),
	}
}

// Start launches the Python AI brain service as a subprocess.
// It polls GET /health until the service is ready or the timeout (30s) is exceeded.
func (b *Bridge) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ready {
		return nil
	}

	name, args, err := findPythonCommand(b.port)
	if err != nil {
		return fmt.Errorf("python brain not found: %w", err)
	}

	b.cmd = exec.CommandContext(ctx, name, args...)

	// Capture stderr for logging.
	stderr, err := b.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := b.cmd.Start(); err != nil {
		return fmt.Errorf("starting python brain (%s %v): %w", name, args, err)
	}

	// Log stderr in a background goroutine.
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[ai-brain] %s", scanner.Text())
		}
	}()

	// Wait for the service to become ready.
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := b.waitReady(waitCtx); err != nil {
		// Clean up the process if it didn't become ready.
		_ = b.stopLocked()
		return fmt.Errorf("brain did not become ready: %w", err)
	}

	b.ready = true
	return nil
}

// Stop gracefully shuts down the Python brain.
// It sends SIGTERM first, then SIGKILL after 5 seconds if needed.
func (b *Bridge) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopLocked()
}

func (b *Bridge) stopLocked() error {
	b.ready = false

	if b.cmd == nil || b.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM.
	if err := b.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited.
		return nil
	}

	// Wait up to 5 seconds for graceful shutdown.
	done := make(chan error, 1)
	go func() {
		done <- b.cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		// Force kill.
		_ = b.cmd.Process.Kill()
		<-done
		return nil
	}
}

// IsReady returns whether the brain service is ready to accept requests.
func (b *Bridge) IsReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ready
}

// Address returns the base URL of the running brain service.
func (b *Bridge) Address() string {
	return b.addr
}

// WaitReady polls the /health endpoint until the service responds or the timeout is exceeded.
func (b *Bridge) WaitReady(ctx context.Context, timeout time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return b.waitReady(waitCtx)
}

func (b *Bridge) waitReady(ctx context.Context) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 2 * time.Second}
	url := b.addr + "/health"

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for brain at %s: %w", url, ctx.Err())
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

// findPythonCommand determines how to launch the Python brain.
// It checks in order:
//  1. "probex-brain" (installed as a standalone command)
//  2. "python3 -m probex_brain.server"
//  3. "python -m probex_brain.server"
func findPythonCommand(port int) (string, []string, error) {
	portStr := fmt.Sprintf("%d", port)

	// Check for standalone command.
	if path, err := exec.LookPath("probex-brain"); err == nil {
		return path, []string{"--port", portStr}, nil
	}

	// Check for python3.
	if path, err := exec.LookPath("python3"); err == nil {
		return path, []string{"-m", "probex_brain.server", "--port", portStr}, nil
	}

	// Check for python.
	if path, err := exec.LookPath("python"); err == nil {
		return path, []string{"-m", "probex_brain.server", "--port", portStr}, nil
	}

	return "", nil, fmt.Errorf("no python executable found; install Python or the probex-brain package")
}
