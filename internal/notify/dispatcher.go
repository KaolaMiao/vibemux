package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/lazyvibe/vibemux/internal/model"
)

// EventType represents a notification event type.
type EventType string

const (
	EventNotify        EventType = "notify"
	EventInputRequired EventType = "input_required"
	EventTaskCompleted EventType = "task_completed"
	EventError         EventType = "error"
)

// Event describes a notification event.
type Event struct {
	ProjectID   string
	ProjectName string
	Type        EventType
	Title       string
	Message     string
	Timestamp   time.Time
}

// Dispatcher sends notifications to configured channels.
type Dispatcher struct {
	client *http.Client
}

// NewDispatcher creates a Dispatcher with sensible defaults.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Dispatch sends a notification event using the given config.
func (d *Dispatcher) Dispatch(ctx context.Context, cfg model.NotificationConfig, event Event) {
	title := strings.TrimSpace(event.Title)
	if title == "" {
		if event.ProjectName != "" {
			title = event.ProjectName
		} else {
			title = "VibeMux"
		}
	}
	message := strings.TrimSpace(event.Message)
	if message == "" {
		message = string(event.Type)
	}
	if len(message) > 800 {
		message = message[:800] + "..."
	}

	if cfg.Desktop {
		_ = beeep.Notify(title, message, "")
	}

	if cfg.WebhookURL != "" {
		payload := map[string]any{
			"project":   event.ProjectName,
			"projectId": event.ProjectID,
			"event":     event.Type,
			"title":     title,
			"message":   message,
			"timestamp": event.Timestamp.Unix(),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := d.client.Do(req)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}
