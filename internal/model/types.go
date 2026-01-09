// Package model defines core data structures for VibeMux.
package model

// DriverType represents the type of driver used to launch AI agent.
type DriverType string

const (
	// DriverNative uses the native claude command directly.
	DriverNative DriverType = "native"
	// DriverCCR uses Claude Code Router for API proxy.
	DriverCCR DriverType = "ccr"
	// DriverCustom allows arbitrary shell commands.
	DriverCustom DriverType = "custom"
)

// AutoApproveLevel defines the level of automatic approval for operations.
type AutoApproveLevel string

const (
	// AutoApproveNone requires manual confirmation for all operations.
	AutoApproveNone AutoApproveLevel = "none"
	// AutoApproveSafe auto-approves read operations and tests.
	AutoApproveSafe AutoApproveLevel = "safe"
	// AutoApproveVibe auto-approves file writes and installs (default).
	AutoApproveVibe AutoApproveLevel = "vibe"
	// AutoApproveYolo auto-approves all shell commands (dangerous).
	AutoApproveYolo AutoApproveLevel = "yolo"
)

// SessionStatus represents the current state of a PTY session.
type SessionStatus string

const (
	// SessionStatusIdle indicates the session is not running.
	SessionStatusIdle SessionStatus = "idle"
	// SessionStatusRunning indicates the session is active.
	SessionStatusRunning SessionStatus = "running"
	// SessionStatusStopped indicates the session has been stopped.
	SessionStatusStopped SessionStatus = "stopped"
	// SessionStatusError indicates the session encountered an error.
	SessionStatusError SessionStatus = "error"
)

// NotificationConfig holds notification settings for a profile.
type NotificationConfig struct {
	// Desktop enables desktop notifications via system APIs.
	Desktop bool `json:"desktop"`
	// WebhookURL is the optional URL to send webhook notifications.
	WebhookURL string `json:"webhook_url,omitempty"`
}
