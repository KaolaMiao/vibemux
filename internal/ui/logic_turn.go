package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/model"
)

// Turn Logic & Auto-Turn Mechanism

// parseTurnSequence parses a sequence string like "0,1,2,1,2" or "0-3" into a list of terminal IDs.
// It maps the indices (0-based) to the actual Project IDs from the grid.
func (a *App) parseTurnSequence(input string, gridIDs []string) []string {
	if input == "" {
		// Default: Round Robin 0..N
		return gridIDs
	}

	parts := strings.Split(input, ",")
	var resultIDs []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Range: "0-2"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil && start <= end {
					for i := start; i <= end; i++ {
						if i >= 0 && i < len(gridIDs) {
							resultIDs = append(resultIDs, gridIDs[i])
						}
					}
				}
			}
		} else {
			// Single index: "1"
			idx, err := strconv.Atoi(part)
			if err == nil {
				if idx >= 0 && idx < len(gridIDs) {
					resultIDs = append(resultIDs, gridIDs[idx])
				}
			}
		}
	}

	if len(resultIDs) == 0 {
		return gridIDs
	}
	return resultIDs
}

// initAutoTurn initializes the turn sequence data but does NOT start the turn.
// This allows for manual confirmation before starting.
func (a *App) initAutoTurn(sequenceStr string) {
	a.turnSequence = a.parseTurnSequence(sequenceStr, a.gridOrder())
	a.currentSeqIndex = 0
	a.autoTurnEnabled = false // Default to paused/manual start
	a.autoTurnCountdown = 10 // User requested 10s default
	a.updateTurnStatus()
	a.statusBar.SetMessage("Roles assigned. Press Alt+A to start auto-turn.", false)
}

// startAutoTurn initializes and STARTS the first turn immediately (Legacy/Manual usage).
func (a *App) startAutoTurn(sequenceStr string) tea.Cmd {
	a.initAutoTurn(sequenceStr)
	a.autoTurnEnabled = true
	return a.sendCurrentTurn()
}

// sendNextTurn advances to the next turn in the sequence.
func (a *App) sendNextTurn() tea.Cmd {
	if len(a.turnSequence) == 0 {
		return nil
	}
	
	a.currentSeqIndex++
	
	// Check if sequence is finished
	if a.currentSeqIndex >= len(a.turnSequence) {
		a.autoTurnEnabled = false
		a.updateTurnStatus()
		a.statusBar.SetMessage("Auto-Turn Sequence Completed", false)
		return nil
	}

	a.updateTurnStatus()
	return a.sendCurrentTurn()
}

// sendCurrentTurn sends the "Your Turn" signal to the current agent in the sequence.
func (a *App) sendCurrentTurn() tea.Cmd {
	if len(a.turnSequence) == 0 || a.currentSeqIndex >= len(a.turnSequence) {
		return nil
	}

	targetID := a.turnSequence[a.currentSeqIndex]
	a.activeTermID = targetID // Switch focus to the active agent
	a.updateFocusStyles()
	
	// Reset Timeout Tracking
	a.currentTurnStartTime = time.Now()

	cmd := func() tea.Msg {
		session, ok := a.engine.GetSession(targetID)
		if !ok || session.Status() != model.SessionStatusRunning {
			return nil
		}

		// Send "Your Turn" command
		// Use \r (Carriage Return) to submit the command in PTY
		msg := fmt.Sprintf("[SYSTEM] 你的回合已到。请读取文件 %s 并执行输出。", a.turnFilename)
		session.Write([]byte(msg))
		time.Sleep(200 * time.Millisecond) // Delay for terminal to process
		session.Write([]byte("\r")) // Submit with Enter
		
		return nil
	}
	
	// Schedule a timeout check (e.g., 2 minutes)
	timeoutCmd := tea.Tick(2 * time.Minute, func(t time.Time) tea.Msg {
		return AutoTurnTimeoutMsg{TargetID: targetID, StartTime: a.currentTurnStartTime}
	})
	
	return tea.Batch(cmd, timeoutCmd)
}

type AutoTurnTimeoutMsg struct {
	TargetID  string
	StartTime time.Time
}

// toggleAutoTurn toggles the auto-turn feature. 
// If enabling from a stopped state, it triggers the current turn.
func (a *App) toggleAutoTurn() tea.Cmd {
	a.autoTurnEnabled = !a.autoTurnEnabled
	status := "OFF"
	cmd := tea.Cmd(nil)
	
	if a.autoTurnEnabled {
		status = "ON"
		// If we are just starting (index 0) or resuming, trigger the current turn
		if len(a.turnSequence) > 0 && a.currentSeqIndex < len(a.turnSequence) {
			cmd = a.sendCurrentTurn()
		}
	}
	
	a.updateTurnStatus()
	a.statusBar.SetMessage(fmt.Sprintf("Auto-Turn: %s", status), false)
	return cmd
}

// updateTurnStatus updates the status bar with current turn info.
func (a *App) updateTurnStatus() {
	if !a.autoTurnEnabled {
		a.statusBar.SetTurnInfo("")
		return
	}
	
	total := len(a.turnSequence)
	if total == 0 {
		return
	}
	
	// 1-based index for display
	current := a.currentSeqIndex + 1
	if current > total {
		current = total
	}
	
	info := fmt.Sprintf("SEQ: %d/%d (Next: %s)", current, total, a.turnSequence[a.currentSeqIndex])
	
	if a.autoTurnCountdown > 0 {
		info += fmt.Sprintf(" [Auto in %ds]", a.autoTurnCountdown)
	}

	a.statusBar.SetTurnInfo(info)
}
