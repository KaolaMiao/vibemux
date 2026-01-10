package ui

import (
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/notify"
)

const (
	oscTailLimit  = 2048
	textTailLimit = 4096
)

var (
	reInputRequired   = regexp.MustCompile(`(?i)(\[[yY]/[nN]\]|\(y/n\)|\bpress enter\b|\brequires your (approval|confirmation)\b|\bneed(s)? your input\b)`)
	reCompleted       = regexp.MustCompile(`(?i)(\btask (finished|complete)\b|\bcost:\s*\$)`)
	reError           = regexp.MustCompile(`(?i)(\berror:\b|context window exceeded|traceback)`)
	reNotifyLine      = regexp.MustCompile(`(?i)^\s*(?:\[notify\]|notify(?:ication)?)[\s:：-]+(.+)$`)
	reVibeNotify      = regexp.MustCompile(`(?i)^\s*vibecode(?:\s+notify)?[\s:：-]+(.+)$`)
	reCommandApproval = regexp.MustCompile(`(?i)(\bdo you want to run\b|\brun (these|the) commands?\b|\bexecute (these|the) commands?\b|\bcommand\b.*\[[yY]/[nN]\])`)
)

type outputWatcher struct {
	oscTail          string
	textTail         string
	lastEvents       map[string]time.Time
	pendingAutoReply string
	pendingAutoTurn  bool
}

func newOutputWatcher() *outputWatcher {
	return &outputWatcher{
		lastEvents: make(map[string]time.Time),
	}
}

func (w *outputWatcher) Process(project *model.Project, profile *model.Profile, data []byte) []notify.Event {
	if len(data) == 0 {
		return nil
	}
	now := time.Now()
	var events []notify.Event

	input := w.oscTail + string(data)
	oscEvents, tail := extractOscNotifications(input)
	w.oscTail = trimTail(tail, oscTailLimit)
	for _, ev := range oscEvents {
		ev.ProjectID = project.ID
		ev.ProjectName = project.Name
		ev.Timestamp = now
		if w.shouldFire(ev) {
			events = append(events, ev)
		}
	}

	plain := ansi.Strip(string(data))
	if plain != "" {
		plain = strings.ReplaceAll(plain, "\r", "\n")
		combined := w.textTail + plain
		
		// NOTE: Auto-turn signal detection removed - using manual control now

		w.textTail = trimTail(combined, textTailLimit)
		lines := tailLines(combined, 12)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if shouldAutoApprove(profile) && w.pendingAutoReply == "" {
				if reInputRequired.MatchString(line) && reCommandApproval.MatchString(line) {
					if w.shouldAutoReply(line) {
						w.pendingAutoReply = "y\r"
					}
				}
			}
			if reInputRequired.MatchString(line) {
				events = appendEventIfNew(events, w, notify.Event{
					Type:    notify.EventInputRequired,
					Title:   "Input required",
					Message: line,
				}, project, now)
				continue
			}
			if reError.MatchString(line) {
				events = appendEventIfNew(events, w, notify.Event{
					Type:    notify.EventError,
					Title:   "Error",
					Message: line,
				}, project, now)
				continue
			}
			if reCompleted.MatchString(line) {
				events = appendEventIfNew(events, w, notify.Event{
					Type:    notify.EventTaskCompleted,
					Title:   "Task completed",
					Message: line,
				}, project, now)
				continue
			}
			if m := reVibeNotify.FindStringSubmatch(line); len(m) == 2 {
				events = appendEventIfNew(events, w, notify.Event{
					Type:    notify.EventNotify,
					Title:   "Notification",
					Message: strings.TrimSpace(m[1]),
				}, project, now)
				continue
			}
			if m := reNotifyLine.FindStringSubmatch(line); len(m) == 2 {
				events = appendEventIfNew(events, w, notify.Event{
					Type:    notify.EventNotify,
					Title:   "Notification",
					Message: strings.TrimSpace(m[1]),
				}, project, now)
			}
		}
	}

	if strings.ContainsRune(input, '\a') {
		events = appendEventIfNew(events, w, notify.Event{
			Type:    notify.EventNotify,
			Title:   "Bell",
			Message: "Terminal bell",
		}, project, now)
	}

	return events
}

func shouldAutoApprove(profile *model.Profile) bool {
	if profile == nil {
		return false
	}
	switch profile.AutoApprove {
	case model.AutoApproveVibe, model.AutoApproveYolo:
		return true
	default:
		return false
	}
}

func (w *outputWatcher) shouldAutoReply(line string) bool {
	if w.lastEvents == nil {
		w.lastEvents = make(map[string]time.Time)
	}
	key := "autoapprove|" + line
	const cooldown = 8 * time.Second
	if last, ok := w.lastEvents[key]; ok && time.Since(last) < cooldown {
		return false
	}
	w.lastEvents[key] = time.Now()
	return true
}

func (w *outputWatcher) ConsumeAutoReply() string {
	if w.pendingAutoReply == "" {
		return ""
	}
	reply := w.pendingAutoReply
	w.pendingAutoReply = ""
	return reply
}

func (w *outputWatcher) ConsumeAutoTurnSignal() bool {
	if w.pendingAutoTurn {
		w.pendingAutoTurn = false
		return true
	}
	return false
}

func appendEventIfNew(events []notify.Event, w *outputWatcher, ev notify.Event, project *model.Project, ts time.Time) []notify.Event {
	ev.ProjectID = project.ID
	ev.ProjectName = project.Name
	ev.Timestamp = ts
	if w.shouldFire(ev) {
		return append(events, ev)
	}
	return events
}

func (w *outputWatcher) shouldFire(ev notify.Event) bool {
	if w.lastEvents == nil {
		w.lastEvents = make(map[string]time.Time)
	}
	key := string(ev.Type) + "|" + ev.Title + "|" + ev.Message
	const cooldown = 12 * time.Second
	if last, ok := w.lastEvents[key]; ok && time.Since(last) < cooldown {
		return false
	}
	w.lastEvents[key] = time.Now()
	if len(w.lastEvents) > 128 {
		for k, v := range w.lastEvents {
			if time.Since(v) > cooldown {
				delete(w.lastEvents, k)
			}
		}
	}
	return true
}

func extractOscNotifications(input string) ([]notify.Event, string) {
	var events []notify.Event
	i := 0
	for i < len(input) {
		if input[i] != 0x1b || i+1 >= len(input) || input[i+1] != ']' {
			i++
			continue
		}
		start := i + 2
		end, termLen := oscTerminator(input, start)
		if end == -1 {
			break
		}
		content := input[start:end]
		events = append(events, decodeOscNotification(content)...)
		i = end + termLen
	}
	return events, input[i:]
}

func oscTerminator(input string, start int) (int, int) {
	if start >= len(input) {
		return -1, 0
	}
	bel := strings.IndexByte(input[start:], 0x07)
	st := strings.Index(input[start:], "\x1b\\")
	if bel == -1 && st == -1 {
		return -1, 0
	}
	if bel != -1 && (st == -1 || bel < st) {
		return start + bel, 1
	}
	return start + st, 2
}

func decodeOscNotification(content string) []notify.Event {
	parts := strings.Split(content, ";")
	if len(parts) == 0 {
		return nil
	}
	switch strings.TrimSpace(parts[0]) {
	case "9":
		msg := strings.TrimSpace(strings.Join(parts[1:], ";"))
		if msg == "" {
			return nil
		}
		return []notify.Event{{
			Type:    notify.EventNotify,
			Title:   "Notification",
			Message: msg,
		}}
	case "777":
		if len(parts) >= 3 && strings.TrimSpace(parts[1]) == "notify" {
			title := strings.TrimSpace(parts[2])
			msg := ""
			if len(parts) > 3 {
				msg = strings.TrimSpace(strings.Join(parts[3:], ";"))
			}
			if msg == "" && title == "" {
				return nil
			}
			return []notify.Event{{
				Type:    notify.EventNotify,
				Title:   title,
				Message: msg,
			}}
		}
	}
	return nil
}

func tailLines(s string, max int) []string {
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func trimTail(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[len(s)-limit:]
}
