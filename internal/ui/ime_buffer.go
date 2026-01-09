package ui

import (
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// IMEBuffer handles input buffering for IME (Input Method Editor) compatibility.
// It buffers potential IME preedit characters (ASCII letters) and flushes them
// either when a timeout occurs or when a non-ASCII character is received.
type IMEBuffer struct {
	buffer    []rune
	lastInput time.Time
	timeout   time.Duration
	targetID  string // The session ID to send input to
}

// IMEFlushMsg is sent when the IME buffer timeout expires.
type IMEFlushMsg struct {
	TargetID string
}

// NewIMEBuffer creates a new IME buffer with the default timeout.
func NewIMEBuffer() *IMEBuffer {
	return &IMEBuffer{
		buffer:  make([]rune, 0, 32),
		timeout: 100 * time.Millisecond, // 100ms delay for IME composition
	}
}

// SetTimeout sets the buffer flush timeout.
func (b *IMEBuffer) SetTimeout(d time.Duration) {
	b.timeout = d
}

// SetTarget sets the target session ID for input.
func (b *IMEBuffer) SetTarget(id string) {
	b.targetID = id
}

// ProcessRunes handles incoming runes and determines whether to buffer or flush.
// Returns:
//   - output: bytes to send immediately (nil if buffering)
//   - cmd: a tea.Cmd for scheduling flush timeout (nil if not needed)
//   - shouldFlushFirst: true if buffered content should be flushed before output
func (b *IMEBuffer) ProcessRunes(runes []rune) (output []byte, cmd tea.Cmd, shouldFlushFirst bool) {
	if len(runes) == 0 {
		return nil, nil, false
	}

	// Check if input contains any non-ASCII characters (likely IME output)
	hasNonASCII := false
	for _, r := range runes {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}

	// If we receive non-ASCII (e.g., Chinese characters), this is likely
	// the final IME composition result - clear buffer and send only this
	if hasNonASCII {
		b.Clear()
		return []byte(string(runes)), nil, false
	}

	// Check if this looks like IME preedit input (single ASCII letter)
	if b.isLikelyIMEPreedit(runes) {
		b.buffer = append(b.buffer, runes...)
		b.lastInput = time.Now()
		// Schedule a flush timeout
		cmd = b.scheduleFlush()
		return nil, cmd, false
	}

	// For other input (punctuation, numbers, etc.), flush buffer first then send
	if len(b.buffer) > 0 {
		return []byte(string(runes)), nil, true
	}

	return []byte(string(runes)), nil, false
}

// isLikelyIMEPreedit checks if the input looks like IME preedit characters.
// IME preedit typically consists of lowercase ASCII letters (pinyin, etc.)
func (b *IMEBuffer) isLikelyIMEPreedit(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}

	// Single lowercase ASCII letter is likely IME preedit
	if len(runes) == 1 {
		r := runes[0]
		return unicode.IsLetter(r) && r >= 'a' && r <= 'z'
	}

	// Multiple characters: check if all are lowercase ASCII letters
	for _, r := range runes {
		if !unicode.IsLetter(r) || r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// scheduleFlush returns a command that will send an IMEFlushMsg after the timeout.
func (b *IMEBuffer) scheduleFlush() tea.Cmd {
	targetID := b.targetID
	timeout := b.timeout
	return tea.Tick(timeout, func(t time.Time) tea.Msg {
		return IMEFlushMsg{TargetID: targetID}
	})
}

// Flush returns all buffered content and clears the buffer.
func (b *IMEBuffer) Flush() []byte {
	if len(b.buffer) == 0 {
		return nil
	}
	output := []byte(string(b.buffer))
	b.buffer = b.buffer[:0]
	return output
}

// Clear empties the buffer without returning content.
func (b *IMEBuffer) Clear() {
	b.buffer = b.buffer[:0]
}

// HasContent returns true if the buffer has pending content.
func (b *IMEBuffer) HasContent() bool {
	return len(b.buffer) > 0
}

// Content returns a copy of the current buffer content.
func (b *IMEBuffer) Content() string {
	return string(b.buffer)
}

// ShouldFlush checks if enough time has passed since the last input.
func (b *IMEBuffer) ShouldFlush() bool {
	if len(b.buffer) == 0 {
		return false
	}
	return time.Since(b.lastInput) >= b.timeout
}
