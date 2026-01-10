package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/runtime"
)

// cycleDispatchMode cycles through dispatch modes: Solo -> Broadcast -> Chain -> Solo.
func (a *App) cycleDispatchMode() {
	switch a.dispatchMode {
	case DispatchModeSolo:
		a.dispatchMode = DispatchModeBroadcast
	case DispatchModeBroadcast:
		a.dispatchMode = DispatchModeChain
		// Initialize chain context if not present or if we want a new one on mode switch?
		// For now, let's create a new one every time we enter chain mode IF one doesn't exist
		// or maybe we keep it until explicitly cleared?
		// Setting it to nil when leaving mode might be better for "Toggle" behavior,
		// but user might want to switch out to check something and come back.
		// Let's create one if nil.
		if a.chainContext == nil {
			id := fmt.Sprintf("%d", time.Now().Unix())
			dir := filepath.Join(a.configDir, "chain")
			ctx, err := runtime.NewChainContext(id, "Chain Session "+id, dir)
			if err == nil {
				a.chainContext = ctx
				// Autosave immediately?
				_ = ctx.Save()
			}
		}
	case DispatchModeChain:
		a.dispatchMode = DispatchModeSolo

	}
	a.updateFocusStyles()
}

// broadcastInput sends input to all running sessions.
func (a *App) broadcastInput(data []byte) {
	sessions := a.engine.ListSessions()
	for _, s := range sessions {
		if s.Status() == model.SessionStatusRunning {
			s.Write(data)
		}
	}
}
