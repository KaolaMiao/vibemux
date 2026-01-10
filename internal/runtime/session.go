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
	id        string
	cmd       *exec.Cmd
	pCmd      *pty.Cmd // Active PTY command
	ptmx      pty.Pty
	output    chan []byte
	done      chan struct{}
	closeOnce sync.Once // 确保 done channel 只关闭一次，防止 panic
	status    model.SessionStatus
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	exitErr   error
	buffer    *RingBuffer // Output history buffer
	initialRows uint16
	initialCols uint16
}

// NewPTYSession creates a new PTY session.
func NewPTYSession(id string, cmd *exec.Cmd) *PTYSession {
	return &PTYSession{
		id:          id,
		cmd:         cmd,
		output:      make(chan []byte, 512), // 缓冲通道，增大容量减少高输出时丢包
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

// readLoop 持续从 PTY 读取输出并发送到 output channel。
// 使用非阻塞发送策略，高负载时丢弃旧数据以保持实时性。
func (s *PTYSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.done:
			return
		default:
			n, err := s.ptmx.Read(buf)
			if err != nil {
				// PTY 读取错误，可能是 EOF（进程结束）或其他 I/O 错误
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

				// 存储到环形缓冲区以保存历史记录
				s.buffer.Write(data)

				// 非阻塞发送到 output channel
				// 策略：优先保证最新数据，如果 channel 满了则丢弃最旧的数据
				select {
				case s.output <- data:
					// 发送成功
				default:
					// Channel 已满，尝试丢弃一个旧数据后重试
					select {
					case <-s.output:
						// 成功丢弃一个旧数据
					default:
						// Channel 已空（可能同时被消费），继续尝试发送
					}
					// 再次尝试非阻塞发送
					select {
					case s.output <- data:
					default:
						// 仍然失败，丢弃当前数据（极端情况）
						// 数据已保存在 RingBuffer 中，不会完全丢失
					}
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

// Stop 终止 PTY 进程。
// 使用 sync.Once 确保 done channel 只关闭一次，防止重复调用导致 panic。
func (s *PTYSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != model.SessionStatusRunning {
		return nil
	}

	// 使用 sync.Once 安全关闭 done channel
	s.closeOnce.Do(func() {
		close(s.done)
	})

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
    if err := s.ptmx.Resize(int(cols), int(rows)); err != nil {
        return err
    }
    
    // Send ANSI escape sequence to force terminal redraw
    // CSI 8 ; rows ; cols t = Resize window to rows x cols (xterm)
    // Some terminals may ignore this, but sending it won't hurt.
    // Also send a simple query that forces apps to update their size.
    // We use DSR (Device Status Report, CSI 6 n) to prompt a response.
    _, _ = s.ptmx.Write([]byte("\x1b[6n"))
    return nil
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
