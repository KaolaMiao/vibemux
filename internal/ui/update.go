package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/model"
)

// Update handles all messages for the application.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// If a dialog is open, only intercept key input; allow other messages through.
	if a.dialogMode != DialogNone {
		if _, ok := msg.(tea.KeyMsg); ok {
			return a.handleDialogUpdate(msg)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		if key.Matches(msg, a.keys.ModeToggle) {
			a.toggleInputMode()
			return a, nil
		}

		if a.inputMode != InputModeTerminal {
			if key.Matches(msg, a.keys.Tab) {
				a.cycleFocus()
				return a, nil
			}
			if key.Matches(msg, a.keys.ShiftTab) {
				a.cycleFocusReverse()
				return a, nil
			}
		}

		if a.inputMode == InputModeTerminal && a.focus == FocusTerminal {
			return a.handleTerminalKeys(msg)
		}

		if key.Matches(msg, a.keys.Quit) {
			a.quitting = true
			a.engine.CloseAll()
			return a, tea.Quit
		}

		if key.Matches(msg, a.keys.Profiles) {
			a.showProfileManager()
			return a, nil
		}

		if a.focus == FocusTerminal {
			if a.handlePaneNavigation(msg) {
				return a, nil
			}
		}

		// Handle pane-specific keys
		return a.handlePaneKeys(msg)

	case ProjectsLoadedMsg:
		if msg.Err == nil {
			a.projects = msg.Projects
			runningIDs := make(map[string]bool)
			for _, s := range a.engine.ListSessions() {
				if s.Status() == model.SessionStatusRunning {
					runningIDs[s.ID()] = true
				}
			}
			a.projectList.SetProjects(a.projects, runningIDs)
		} else {
			a.statusBar.SetMessage("Error loading projects: "+msg.Err.Error(), true)
		}
		return a, nil

	case ProfilesLoadedMsg:
		if msg.Err == nil {
			a.profiles = msg.Profiles
			a.updateAddDialogProfiles()
			a.profileList.SetProfiles(a.profiles)
			a.projectList.SetProfiles(a.profiles)
		} else {
			a.statusBar.SetMessage("Error loading profiles: "+msg.Err.Error(), true)
		}
		return a, nil

	case ProjectCreatedMsg:
		a.statusBar.SetMessage("Project added: "+msg.Project.Name, false)
		return a, a.loadProjects()

	case ProfileSavedMsg:
		a.upsertProfileInMemory(msg.Profile)
		if msg.IsNew {
			a.statusBar.SetMessage("Profile added: "+msg.Profile.Name, false)
		} else {
			a.statusBar.SetMessage("Profile updated: "+msg.Profile.Name, false)
		}
		if !msg.IsNew {
			var restartCmds []tea.Cmd
			for i := range a.projects {
				p := a.projects[i]
				usesProfile := p.ProfileID == msg.Profile.ID
				if !usesProfile && p.ProfileID == "" && msg.Profile.IsDefault {
					usesProfile = true
				}
				if !usesProfile {
					continue
				}
				if session, ok := a.engine.GetSession(p.ID); ok && session.Status() == model.SessionStatusRunning {
					_ = a.engine.CloseSession(p.ID)
					if inst, ok := a.terminals[p.ID]; ok {
						inst.Terminal.SetStatus(model.SessionStatusStopped)
						inst.Terminal.Clear()
					}
					a.projectList.SetRunning(p.ID, false)
					a.sessionTabs.SetTabStatus(p.ID, model.SessionStatusStopped)
					restartCmds = append(restartCmds, a.startSession(&p))
				}
			}
			if len(restartCmds) > 0 {
				return a, tea.Batch(append(restartCmds, a.loadProfiles())...)
			}
		}
		return a, a.loadProfiles()

	case ProfileDeletedMsg:
		a.statusBar.SetMessage("Profile deleted", false)
		return a, a.loadProfiles()

	case SessionStartedMsg:
		a.setActivePaneByProject(msg.ProjectID)
		a.outputWatchers[msg.ProjectID] = newOutputWatcher()
		// Update terminal status
		if inst, ok := a.terminals[msg.ProjectID]; ok {
			inst.Terminal.SetStatus(model.SessionStatusRunning)
			if session, ok := a.engine.GetSession(msg.ProjectID); ok {
				inst.Terminal.BindWriter(session)
				cols, rows := inst.Terminal.PTYSize()
				if cols > 0 && rows > 0 {
					_ = session.Resize(uint16(rows), uint16(cols))
				}
			}
		}
		// Update project list
		a.projectList.SetRunning(msg.ProjectID, true)
		// Update session tabs
		a.sessionTabs.SetTabStatus(msg.ProjectID, model.SessionStatusRunning)
		a.statusBar.SetMessage("Session started", false)
		// Start listening for output
		return a, a.waitForOutput(msg.ProjectID)

	case SessionOutputMsg:
		// Update the specific terminal instance
		if inst, ok := a.terminals[msg.ProjectID]; ok {
			inst.Terminal.AppendOutput(msg.Data)
		}
		var notifyCmd tea.Cmd
		if project := a.findProjectByID(msg.ProjectID); project != nil {
			watcher, ok := a.outputWatchers[msg.ProjectID]
			if !ok || watcher == nil {
				watcher = newOutputWatcher()
				a.outputWatchers[msg.ProjectID] = watcher
			}
			profile := a.profileForProject(project)
			events := watcher.Process(project, profile, msg.Data)
			notifyCmd = a.dispatchNotifications(profile, events)
			if reply := watcher.ConsumeAutoReply(); reply != "" {
				if session, ok := a.engine.GetSession(msg.ProjectID); ok && session.Status() == model.SessionStatusRunning {
					session.Write([]byte(reply))
				}
			}
		}
		// Mark tab as having new content if not active
		if msg.ProjectID != a.activeTermID {
			a.sessionTabs.MarkTabHasNew(msg.ProjectID)
		}
		// Continue listening
		return a, tea.Batch(a.waitForOutput(msg.ProjectID), notifyCmd)

	case SessionStoppedMsg:
		if inst, ok := a.terminals[msg.ProjectID]; ok {
			inst.Terminal.SetStatus(model.SessionStatusStopped)
			inst.Terminal.UnbindWriter()
		}
		delete(a.outputWatchers, msg.ProjectID)
		a.projectList.SetRunning(msg.ProjectID, false)
		a.sessionTabs.SetTabStatus(msg.ProjectID, model.SessionStatusStopped)
		if msg.Err != nil {
			a.statusBar.SetMessage("Session error: "+msg.Err.Error(), true)
		} else {
			a.statusBar.SetMessage("Session ended", false)
		}
		return a, nil

	case ErrorMsg:
		a.statusBar.SetMessage("Error: "+msg.Err.Error(), true)
		return a, nil

	case IMEFlushMsg:
		// Handle IME buffer flush timeout
		if a.inputMode == InputModeTerminal && a.activeTermID == msg.TargetID {
			if a.imeBuffer.HasContent() {
				if session, ok := a.engine.GetSession(a.activeTermID); ok && session.Status() == model.SessionStatusRunning {
					session.Write(a.imeBuffer.Flush())
				} else {
					a.imeBuffer.Clear()
				}
			}
		}
		return a, nil
	}

	// Update focused component
	switch a.focus {
	case FocusProjects:
		var cmd tea.Cmd
		a.projectList, cmd = a.projectList.Update(msg)
		cmds = append(cmds, cmd)
	case FocusTerminal:
		if inst, ok := a.terminals[a.activeTermID]; ok {
			var cmd tea.Cmd
			inst.Terminal, cmd = inst.Terminal.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

// handleDialogUpdate handles input when a dialog is open.
func (a App) handleDialogUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch a.dialogMode {
	case DialogAddProject:
		var cmd tea.Cmd
		a.addDialog, cmd = a.addDialog.Update(msg)

		if a.addDialog.IsSubmitted() {
			a.hideDialog()
			return a, a.createProject()
		}
		if a.addDialog.IsCancelled() {
			a.hideDialog()
			return a, nil
		}
		return a, cmd
	case DialogManageProfiles:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "esc":
				a.hideDialog()
				return a, nil
			case "a":
				a.showProfileDialog(nil)
				return a, nil
			case "c":
				a.showSettingsDialog()
				return a, nil
			case "enter", "e":
				if profile := a.profileList.SelectedProfile(); profile != nil {
					a.showProfileDialog(profile)
				}
				return a, nil
			case "d":
				if profile := a.profileList.SelectedProfile(); profile != nil {
					if profile.IsDefault {
						a.statusBar.SetMessage("Cannot delete default profile", true)
						return a, nil
					}
					return a, a.deleteProfile(profile.ID)
				}
				return a, nil
			case "s":
				if profile := a.profileList.SelectedProfile(); profile != nil {
					a.statusBar.SetMessage("Default profile set", false)
					return a, a.setDefaultProfile(profile.ID)
				}
				return a, nil
			}
			a.profileList.HandleKey(keyMsg.String())
		}
		return a, nil
	case DialogEditProfile:
		var cmd tea.Cmd
		a.profileDialog, cmd = a.profileDialog.Update(msg)
		if a.profileDialog.IsSubmitted() {
			profile, isNew, err := a.buildProfileFromDialog()
			if err != nil {
				a.statusBar.SetMessage(err.Error(), true)
				return a, nil
			}
			a.dialogMode = DialogManageProfiles
			a.profileEditID = ""
			return a, a.saveProfile(profile, isNew)
		}
		if a.profileDialog.IsCancelled() {
			a.dialogMode = DialogManageProfiles
			a.profileEditID = ""
			return a, nil
		}
		return a, cmd
	case DialogSettings:
		var cmd tea.Cmd
		a.settingsDialog, cmd = a.settingsDialog.Update(msg)
		if a.settingsDialog.IsSubmitted() {
			values := a.settingsDialog.Values()
			input := ""
			if len(values) > 0 {
				input = values[0]
			}
			rows, cols, err := parseGridSetting(input)
			if err != nil {
				a.statusBar.SetMessage(err.Error(), true)
				return a, nil
			}
			if err := a.updateGridSettings(rows, cols); err != nil {
				a.statusBar.SetMessage("Error saving config: "+err.Error(), true)
				return a, nil
			}
			a.statusBar.SetMessage(fmt.Sprintf("Grid set to %dx%d", rows, cols), false)
			a.dialogMode = DialogManageProfiles
			return a, nil
		}
		if a.settingsDialog.IsCancelled() {
			a.dialogMode = DialogManageProfiles
			return a, nil
		}
		return a, cmd
	case DialogCommand:
		var cmd tea.Cmd
		a.commandDialog, cmd = a.commandDialog.Update(msg)
		if a.commandDialog.IsSubmitted() {
			values := a.commandDialog.Values()
			input := ""
			if len(values) > 0 {
				input = values[0]
			}
			a.hideDialog()
			return a, a.executeCommand(input)
		}
		if a.commandDialog.IsCancelled() {
			a.hideDialog()
			return a, nil
		}
		return a, cmd
	}
	return a, nil
}

// handlePaneKeys processes keyboard input for the focused pane.
func (a App) handlePaneKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.focus {
	case FocusProjects:
		return a.handleProjectsKeys(msg)
	case FocusTerminal:
		return a.handleTerminalControlKeys(msg)
	}
	return a, nil
}

func (a App) handleTerminalControlKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Enter):
		a.enterTerminalMode()
		return a, nil
	case key.Matches(msg, a.keys.Close):
		if a.activeTermID != "" {
			a.closeSession(a.activeTermID)
			a.statusBar.SetMessage("Session closed", false)
		}
		return a, nil
	}
	if inst, ok := a.terminals[a.activeTermID]; ok {
		if inst.Terminal.HandleKey(msg.String()) {
			return a, nil
		}
	}
	return a, nil
}

func (a *App) handlePaneNavigation(msg tea.KeyMsg) bool {
	ids := a.gridOrder()
	if len(ids) == 0 {
		return false
	}
	if !(key.Matches(msg, a.keys.PaneLeft) ||
		key.Matches(msg, a.keys.PaneRight) ||
		key.Matches(msg, a.keys.PaneUp) ||
		key.Matches(msg, a.keys.PaneDown)) {
		return false
	}

	rows, cols := a.gridActiveDims()
	if rows < 1 || cols < 1 {
		return false
	}

	row := 0
	col := 0
	if cols > 0 {
		row = a.activePane / cols
		col = a.activePane % cols
	}

	switch {
	case key.Matches(msg, a.keys.PaneLeft):
		if col > 0 {
			col--
		} else {
			a.focus = FocusProjects
			a.updateFocusStyles()
			return true
		}
	case key.Matches(msg, a.keys.PaneRight):
		if col < cols-1 {
			col++
		}
	case key.Matches(msg, a.keys.PaneUp):
		if row > 0 {
			row--
		}
	case key.Matches(msg, a.keys.PaneDown):
		if row < rows-1 {
			row++
		}
	}

	index := row*cols + col
	if index >= len(ids) {
		return true
	}
	a.setActivePane(index)
	return true
}

// handleProjectsKeys handles keys when project list is focused.
func (a App) handleProjectsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.PaneRight):
		ids := a.gridOrder()
		if len(ids) > 0 {
			a.focus = FocusTerminal
			if a.activeTermID != "" {
				a.setActivePaneByProject(a.activeTermID)
			} else {
				a.setActivePane(0)
			}
			return a, nil
		}
		return a, nil
	case key.Matches(msg, a.keys.Enter):
		// Start/switch to selected project
		project := a.projectList.SelectedProject()
		if project != nil {
			if !a.canOpenPane(project.ID) {
				a.statusBar.SetMessage("Max panes reached for grid layout", true)
				return a, nil
			}
			// Get or create terminal instance
			inst := a.getOrCreateTerminal(project.ID, project.DisplayName())

			// Add to session tabs if not present
			a.sessionTabs.AddTab(project.ID, project.DisplayName(), model.SessionStatusIdle)
			a.setActivePaneByProject(project.ID)
			a.SetSize(a.width, a.height)

			// Check if session already exists
			if session, ok := a.engine.GetSession(project.ID); ok {
				// Session exists, just update terminal status
				inst.Terminal.SetStatus(session.Status())
				if session.Status() == model.SessionStatusRunning {
					// Resume listening for output
					return a, a.waitForOutput(project.ID)
				}
			} else {
				// Start new session
				return a, a.startSession(project)
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.Add):
		a.showAddDialog()
		return a, nil

	case key.Matches(msg, a.keys.Delete):
		// Delete selected project
		project := a.projectList.SelectedProject()
		if project != nil {
			// Close session if running
			a.engine.CloseSession(project.ID)
			// Remove from tabs
			a.sessionTabs.RemoveTab(project.ID)
			// Remove terminal instance
			delete(a.terminals, project.ID)
			delete(a.outputWatchers, project.ID)
			a.normalizeActivePane()
			a.SetSize(a.width, a.height)
			// Delete from store
			if err := a.store.Delete(a.ctx, project.ID); err != nil {
				a.statusBar.SetMessage("Error deleting project: "+err.Error(), true)
			} else {
				a.statusBar.SetMessage("Project deleted", false)
				// Reload projects
				return a, a.loadProjects()
			}
		}
		return a, nil
	case key.Matches(msg, a.keys.Close):
		project := a.projectList.SelectedProject()
		if project != nil {
			a.closeSession(project.ID)
			a.statusBar.SetMessage("Session closed", false)
		}
		return a, nil
	}

	// Let project list handle navigation keys
	a.projectList.HandleKey(msg.String())
	return a, nil
}

// handleTabsKeys handles keys when session tabs are focused.
func (a App) handleTabsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		a.sessionTabs.PrevTab()
		if t := a.sessionTabs.ActiveTab(); t != nil {
			a.activeTermID = t.ID
		}
		a.updateFocusStyles()
	case "right", "l":
		a.sessionTabs.NextTab()
		if t := a.sessionTabs.ActiveTab(); t != nil {
			a.activeTermID = t.ID
		}
		a.updateFocusStyles()
	case "x":
		// Close current tab
		if t := a.sessionTabs.ActiveTab(); t != nil {
			a.closeSession(t.ID)
			a.statusBar.SetMessage("Session closed", false)
		}
	case "enter":
		// Focus on terminal
		a.focus = FocusTerminal
		a.updateFocusStyles()
	}
	return a, nil
}

// handleTerminalKeys handles keys when terminal is focused.
func (a App) handleTerminalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check if we should send input to the session
	if a.activeTermID != "" {
		session, ok := a.engine.GetSession(a.activeTermID)
		if ok && session.Status() == model.SessionStatusRunning {
			// Update IME buffer target
			a.imeBuffer.SetTarget(a.activeTermID)

			// Handle KeyRunes with IME buffering
			if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
				output, cmd, shouldFlushFirst := a.imeBuffer.ProcessRunes(msg.Runes)

				// If we need to flush buffered content first
				if shouldFlushFirst {
					buffered := a.imeBuffer.Flush()
					if len(buffered) > 0 {
						session.Write(buffered)
					}
				}

				// Send immediate output if any
				if len(output) > 0 {
					// Apply Alt modifier if needed
					if msg.Alt {
						output = append([]byte{27}, output...)
					}
					session.Write(output)
				}

				return a, cmd
			}

			// For non-rune keys, flush IME buffer first then send the key
			if a.imeBuffer.HasContent() {
				session.Write(a.imeBuffer.Flush())
			}

			// Send key to PTY
			input := keyToBytes(msg)
			if len(input) > 0 {
				session.Write(input)
				return a, nil
			}
		}
	}

	// Otherwise handle as scroll navigation
	if inst, ok := a.terminals[a.activeTermID]; ok {
		inst.Terminal.HandleKey(msg.String())
	}
	return a, nil
}

// keyToBytes converts a key message to bytes for PTY input.
func keyToBytes(msg tea.KeyMsg) []byte {
	if msg.Type == tea.KeyRunes {
		payload := []byte(string(msg.Runes))
		if msg.Alt && len(payload) > 0 {
			return append([]byte{27}, payload...)
		}
		return payload
	}

	switch msg.Type {
	case tea.KeyUp:
		return arrowSeq("A", msg.Alt, false, false)
	case tea.KeyDown:
		return arrowSeq("B", msg.Alt, false, false)
	case tea.KeyRight:
		return arrowSeq("C", msg.Alt, false, false)
	case tea.KeyLeft:
		return arrowSeq("D", msg.Alt, false, false)
	case tea.KeyShiftUp:
		return arrowSeq("A", msg.Alt, true, false)
	case tea.KeyShiftDown:
		return arrowSeq("B", msg.Alt, true, false)
	case tea.KeyShiftRight:
		return arrowSeq("C", msg.Alt, true, false)
	case tea.KeyShiftLeft:
		return arrowSeq("D", msg.Alt, true, false)
	case tea.KeyCtrlUp:
		return arrowSeq("A", msg.Alt, false, true)
	case tea.KeyCtrlDown:
		return arrowSeq("B", msg.Alt, false, true)
	case tea.KeyCtrlRight:
		return arrowSeq("C", msg.Alt, false, true)
	case tea.KeyCtrlLeft:
		return arrowSeq("D", msg.Alt, false, true)
	case tea.KeyCtrlShiftUp:
		return arrowSeq("A", msg.Alt, true, true)
	case tea.KeyCtrlShiftDown:
		return arrowSeq("B", msg.Alt, true, true)
	case tea.KeyCtrlShiftRight:
		return arrowSeq("C", msg.Alt, true, true)
	case tea.KeyCtrlShiftLeft:
		return arrowSeq("D", msg.Alt, true, true)
	case tea.KeyHome:
		return homeEndSeq("H", msg.Alt, false, false)
	case tea.KeyEnd:
		return homeEndSeq("F", msg.Alt, false, false)
	case tea.KeyShiftHome:
		return homeEndSeq("H", msg.Alt, true, false)
	case tea.KeyShiftEnd:
		return homeEndSeq("F", msg.Alt, true, false)
	case tea.KeyCtrlHome:
		return homeEndSeq("H", msg.Alt, false, true)
	case tea.KeyCtrlEnd:
		return homeEndSeq("F", msg.Alt, false, true)
	case tea.KeyCtrlShiftHome:
		return homeEndSeq("H", msg.Alt, true, true)
	case tea.KeyCtrlShiftEnd:
		return homeEndSeq("F", msg.Alt, true, true)
	case tea.KeyPgUp:
		return tildeSeq("5", msg.Alt, false, false)
	case tea.KeyPgDown:
		return tildeSeq("6", msg.Alt, false, false)
	case tea.KeyCtrlPgUp:
		return tildeSeq("5", msg.Alt, false, true)
	case tea.KeyCtrlPgDown:
		return tildeSeq("6", msg.Alt, false, true)
	case tea.KeyInsert:
		return tildeSeq("2", msg.Alt, false, false)
	case tea.KeyDelete:
		return tildeSeq("3", msg.Alt, false, false)
	}

	var base []byte
	switch msg.Type {
	case tea.KeyEnter:
		base = []byte{'\r'}
	case tea.KeySpace:
		base = []byte{' '}
	case tea.KeyTab:
		base = []byte{'\t'}
	case tea.KeyBackspace:
		base = []byte{127}
	case tea.KeyEscape:
		base = []byte{27}
	}

	if base == nil {
		if msg.Type >= tea.KeyCtrlAt && msg.Type <= tea.KeyCtrlZ {
			base = []byte{byte(msg.Type)}
		} else {
			switch msg.Type {
			case tea.KeyCtrlOpenBracket:
				base = []byte{27}
			case tea.KeyCtrlBackslash:
				base = []byte{28}
			case tea.KeyCtrlCloseBracket:
				base = []byte{29}
			case tea.KeyCtrlCaret:
				base = []byte{30}
			case tea.KeyCtrlUnderscore:
				base = []byte{31}
			case tea.KeyCtrlQuestionMark:
				base = []byte{127}
			}
		}
	}

	if len(base) == 0 {
		return nil
	}
	if msg.Alt {
		return append([]byte{27}, base...)
	}
	return base
}

func arrowSeq(code string, alt, shift, ctrl bool) []byte {
	mod := modifierCode(alt, shift, ctrl)
	if mod == 1 {
		return []byte{27, '[', code[0]}
	}
	return []byte("\x1b[1;" + modString(mod) + code)
}

func homeEndSeq(code string, alt, shift, ctrl bool) []byte {
	mod := modifierCode(alt, shift, ctrl)
	if mod == 1 {
		return []byte{27, '[', code[0]}
	}
	return []byte("\x1b[1;" + modString(mod) + code)
}

func tildeSeq(prefix string, alt, shift, ctrl bool) []byte {
	mod := modifierCode(alt, shift, ctrl)
	if mod == 1 {
		return []byte("\x1b[" + prefix + "~")
	}
	return []byte("\x1b[" + prefix + ";" + modString(mod) + "~")
}

func modifierCode(alt, shift, ctrl bool) int {
	code := 1
	if shift {
		code += 1
	}
	if alt {
		code += 2
	}
	if ctrl {
		code += 4
	}
	return code
}

func modString(code int) string {
	return string('0' + code)
}
