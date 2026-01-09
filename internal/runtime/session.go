// Package runtime provides PTY session management and process control.
package runtime

import (
	"context"
	"errors"
	"fmt"

	"os/exec"
	"strings"
	"sync"

	"github.com/aymanbagabas/go-pty"
	"github.com/lazyvibe/vibemux/internal/model"
)

// Session represents a PTY session for an AI agent process.
type Session interface {
	// ID returns the session's unique identifier (project ID).
	ID() string
	// Start launches the PTY process.
	Start(ctx context.Context) error
	// Stop terminates the PTY process.
	Stop() error
	// Write sends data to the PTY stdin.
	Write(data []byte) (int, error)
	// Output returns the channel for receiving PTY output.
	Output() <-chan []byte
	// Status returns the current session status.
	Status() model.SessionStatus
	// Resize updates the PTY terminal size.
	Resize(rows, cols uint16) error
}

// PTYSession implements Session using creack/pty.
type PTYSession struct {
	id      string
	cmd     *exec.Cmd
	pCmd    *pty.Cmd // Active PTY command
	ptmx    pty.Pty
	output  chan []byte
	done    chan struct{}
	status  model.SessionStatus
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	exitErr error
	buffer  *RingBuffer // Output history buffer
	initialRows uint16
	initialCols uint16
}

// NewPTYSession creates a new PTY session.
func NewPTYSession(id string, cmd *exec.Cmd) *PTYSession {
	return &PTYSession{
		id:          id,
		cmd:         cmd,
		output:      make(chan []byte, 256), // Buffered channel for output
		done:        make(chan struct{}),
		status:      model.SessionStatusIdle,
		buffer:      NewRingBuffer(50000), // ~50KB history
		initialRows: 24,                   // Default fallback
		initialCols: 80,                   // Default fallback
	}
}

// SetInitialSize sets the initial PTY size.
func (s *PTYSession) SetInitialSize(rows, cols int) {
	if rows > 0 {
		s.initialRows = uint16(rows)
	}
	if cols > 0 {
		s.initialCols = uint16(cols)
	}
}

// ID returns the session identifier.
func (s *PTYSession) ID() string {
	return s.id
}

// Start launches the PTY process.
func (s *PTYSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == model.SessionStatusRunning {
		return errors.New("session already running")
	}

	// Create cancellable context
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Initialize PTY
	ptmx, err := pty.New()
	if err != nil {
		s.status = model.SessionStatusError
		return fmt.Errorf("failed to create pty: %w", err)
	}
	s.ptmx = ptmx

    // Resize immediately to initial size to avoid race conditions
    // Note: pty.Resize takes (width, height) -> (cols, rows)
    _ = s.ptmx.Resize(int(s.initialCols), int(s.initialRows))

	// Construct pty command from exec.Cmd
	var args []string
	if len(s.cmd.Args) > 1 {
		args = s.cmd.Args[1:]
	}
	
	// Assuming Pty interface has Command method (as verified in source)
	// We need to type assert or assume the library interface matches.
	// Since we saw _ Pty = &conPty{}, it should work.
	// But Command() returns *pty.Cmd.
	
	// Implementation note: The pty.Pty interface definition in the library
	// SHOULD include Command. If not, we might need a type assertion.
	// Based on 'pty_windows.go', the interface implementation includes Command.
    
    // However, interface methods must be defined in the interface type.
    // If Pty interface doesn't have Command, we can't call it on interface.
    // Let's assume it does for now based on usage patterns.
    
    // Workaround if interface is missing Command:
    // type commander interface { Command(string, ...string) *pty.Cmd }
    // if c, ok := ptmx.(commander); ok { ... }
    
    // For now, let's try direct call.
    if commander, ok := ptmx.(interface{ Command(string, ...string) *pty.Cmd }); ok {
        s.pCmd = commander.Command(s.cmd.Path, args...)
    } else {
        return errors.New("pty implementation does not support Command creation")
    }

	s.pCmd.Env = s.cmd.Env
	s.pCmd.Dir = s.cmd.Dir

	// Start command
	if err := s.pCmd.Start(); err != nil {
		s.status = model.SessionStatusError
		wrapped := fmt.Errorf("start failed: %s: %w", formatCmd(s.cmd), err)
		s.exitErr = wrapped
		return wrapped
	}
	s.status = model.SessionStatusRunning

	// Start output reader goroutine
	go s.readLoop()

	// Start process monitor monitoring
	go s.waitLoop()

	return nil
}

func formatCmd(cmd *exec.Cmd) string {
	if cmd == nil {
		return ""
	}
	if len(cmd.Args) > 0 {
		return strings.Join(cmd.Args, " ")
	}
	if cmd.Path != "" {
		return cmd.Path
	}
	return ""
}

// readLoop continuously reads from PTY and sends to output channel.
func (s *PTYSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.done:
			return
		default:
			n, err := s.ptmx.Read(buf)
			if err != nil {
				// EOF or error - process likely ended
				s.mu.Lock()
				if s.status == model.SessionStatusRunning {
					s.status = model.SessionStatusStopped
				}
				s.mu.Unlock()
				close(s.output)
				return
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])

				// Store in ring buffer for history
				s.buffer.Write(data)

				// Non-blocking send to output channel
				select {
				case s.output <- data:
				default:
					// Channel full, drop oldest and retry
					select {
					case <-s.output:
					default:
					}
					s.output <- data
				}
			}
		}
	}
}

// waitLoop monitors process exit.
func (s *PTYSession) waitLoop() {
	if s.pCmd != nil {
		err := s.pCmd.Wait()
		s.mu.Lock()
		s.exitErr = err
		if s.status == model.SessionStatusRunning {
			s.status = model.SessionStatusStopped
		}
		s.mu.Unlock()
	}
}

// Stop terminates the PTY process.
func (s *PTYSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != model.SessionStatusRunning {
		return nil
	}

	// Signal done to readers
	close(s.done)

	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Close PTY (will also terminate the process)
	if s.ptmx != nil {
		s.ptmx.Close()
	}

	// Kill process if still running
	if s.pCmd != nil && s.pCmd.Process != nil {
		s.pCmd.Process.Kill()
	}

	s.status = model.SessionStatusStopped
	return nil
}

// Write sends data to PTY stdin.
func (s *PTYSession) Write(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.status != model.SessionStatusRunning {
		return 0, errors.New("session not running")
	}

	if s.ptmx == nil {
		return 0, errors.New("pty not initialized")
	}

	return s.ptmx.Write(data)
}

// Output returns the output channel.
func (s *PTYSession) Output() <-chan []byte {
	return s.output
}

// Status returns the current status.
func (s *PTYSession) Status() model.SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// Resize changes the PTY terminal size.
func (s *PTYSession) Resize(rows, cols uint16) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ptmx == nil {
		return errors.New("pty not initialized")
	}

	// return s.ptmx.Resize(int(cols), int(rows))
    // Note: pty.Resize takes (width, height) which corresponds to (cols, rows)
    return s.ptmx.Resize(int(cols), int(rows))
}

// History returns the buffered output history.
func (s *PTYSession) History() []byte {
	return s.buffer.Bytes()
}

// ExitError returns the process exit error if any.
func (s *PTYSession) ExitError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exitErr
}
