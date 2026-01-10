package ui

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"fmt"


	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/app"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/notify"
	"github.com/lazyvibe/vibemux/internal/runtime"
	"github.com/lazyvibe/vibemux/internal/store"
	"github.com/lazyvibe/vibemux/internal/ui/components/chaindialog"
	"github.com/lazyvibe/vibemux/internal/ui/components/configdialog"
	"github.com/lazyvibe/vibemux/internal/ui/components/dialog"
	"github.com/lazyvibe/vibemux/internal/ui/components/filepreview"
	profilelist "github.com/lazyvibe/vibemux/internal/ui/components/profile_list"
	projectlist "github.com/lazyvibe/vibemux/internal/ui/components/project_list"
	"github.com/lazyvibe/vibemux/internal/ui/components/sessiontabs"
	"github.com/lazyvibe/vibemux/internal/ui/components/statusbar"
	"github.com/lazyvibe/vibemux/internal/ui/components/terminal"
	"github.com/lazyvibe/vibemux/internal/ui/keys"
	"github.com/lazyvibe/vibemux/pkg/utils"
)

// FocusArea represents which UI pane has focus.
type FocusArea int

const (
	// FocusProjects is the project list pane.
	FocusProjects FocusArea = iota
	// FocusTerminal is the terminal viewport pane.
	FocusTerminal
)

// InputMode controls whether key input is routed to the PTY or to the UI.
type InputMode int

const (
	InputModeControl InputMode = iota
	InputModeTerminal
)

// DispatchMode controls how input is dispatched to terminals.
type DispatchMode int

const (
	DispatchModeSolo DispatchMode = iota
	DispatchModeBroadcast
	DispatchModeChain
)

const (
	minAppWidth  = 40
	minAppHeight = 10
)

// DialogMode represents the current dialog being shown.
type DialogMode int

const (
	DialogNone DialogMode = iota
	DialogAddProject
	DialogManageProfiles
	DialogEditProfile
	DialogSettings
	DialogCommand
	DialogChainPreview
	DialogAssignRoles
	DialogAssignRolesFile
	DialogFilePreview
)

// TerminalInstance holds data for a single terminal session.
type TerminalInstance struct {
	ProjectID   string
	ProjectName string
	Terminal    terminal.Model
	Content     strings.Builder
}

// App is the main application model.
type App struct {
	// Components
	projectList    projectlist.Model
	profileList    profilelist.Model
	sessionTabs    sessiontabs.Model
	terminals      map[string]*TerminalInstance // Multiple terminal instances
	activeTermID   string                       // Currently displayed terminal
	statusBar      statusbar.Model
	addDialog      dialog.InputDialog
	profileDialog  dialog.InputDialog
	settingsDialog dialog.InputDialog
	commandDialog  dialog.InputDialog
	roleDialog     dialog.InputDialog
	organizerDialog configdialog.Model // Separate complex dialog

	chainDialog    chaindialog.Model
	filePreview    filepreview.Model

	// State
	focus      FocusArea
	dialogMode DialogMode
	width      int
	height     int
	ready      bool
	quitting   bool
	activePane int
	gridRows   int
	gridCols   int
	inputMode    InputMode
	dispatchMode DispatchMode
	imeBuffer    *IMEBuffer // IME input buffer for Chinese input support

	// Data
	projects      []model.Project
	profiles      []model.Profile
	profileEditID string

	tempChainFile string

	// Auto-Turn State
	turnSequence      []string
	currentSeqIndex   int
	autoTurnEnabled   bool
	autoTurnCountdown int // 5s countdown
	turnTopic         string
	turnFilename    string
	currentTurnStartTime time.Time

	configDir string
	config    *app.Config

	// Chain Mode
	chainContext *runtime.ChainContext

	// Dependencies
	store          *store.JSONStore
	engine         *runtime.DefaultEngine
	keys           keys.KeyMap
	ctx            context.Context
	notifier       *notify.Dispatcher
	outputWatchers map[string]*outputWatcher
}

// New creates a new application instance.
func New(s *store.JSONStore, e *runtime.DefaultEngine, cfg *app.Config, configDir string) App {
	rows, cols := sanitizeGridSize(cfg)
	status := statusbar.New()
	status.SetModeLabel("CTRL")
	return App{
		projectList:    projectlist.New(),
		profileList:    profilelist.New(),
		sessionTabs:    sessiontabs.New(),
		filePreview:    filepreview.New(),
		terminals:      make(map[string]*TerminalInstance),
		outputWatchers: make(map[string]*outputWatcher),
		statusBar:      status,
		addDialog: dialog.NewInputDialog("Add Project", []dialog.InputField{
			{Label: "Project Name", Placeholder: "my-awesome-project"},
			{Label: "Project Path", Placeholder: "~/projects/my-project", EnablePathComp: true},
			{Label: "Profile", Placeholder: "default (optional)"},
		}),
		focus:      FocusProjects,
		dialogMode: DialogNone,
		store:      s,
		engine:     e,
		keys:       keys.DefaultKeyMap(),
		ctx:        context.Background(),
		notifier:   notify.NewDispatcher(),
		gridRows:   rows,
		gridCols:   cols,
		inputMode:  InputModeControl,
		imeBuffer:  NewIMEBuffer(),
		configDir:  configDir,
		config:     cfg,
		// Initialize with a default chain session
		chainContext: func() *runtime.ChainContext {
			id := fmt.Sprintf("%d", time.Now().Unix())
			dir := filepath.Join(configDir, "chain")
			ctx, _ := runtime.NewChainContext(id, "Chain Session "+id, dir)
			return ctx
		}(),
	}
}

func sanitizeGridSize(cfg *app.Config) (int, int) {
	rows, cols := 2, 2
	if cfg != nil {
		rows = cfg.GridRows
		cols = cfg.GridCols
	}
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	if rows > 3 {
		rows = 3
	}
	if cols > 3 {
		cols = 3
	}
	return rows, cols
}

func parseGridSetting(input string) (int, int, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return 0, 0, errors.New("grid size is required (4/6/9 or 2x2/2x3/3x3)")
	}

	switch value {
	case "4":
		return 2, 2, nil
	case "6":
		return 2, 3, nil
	case "9":
		return 3, 3, nil
	}

	if strings.Contains(value, "x") {
		parts := strings.SplitN(value, "x", 2)
		if len(parts) == 2 {
			rows, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return 0, 0, errors.New("invalid grid size format")
			}
			cols, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return 0, 0, errors.New("invalid grid size format")
			}
			if rows < 1 || rows > 3 || cols < 1 || cols > 3 {
				return 0, 0, errors.New("grid rows/cols must be between 1 and 3")
			}
			size := rows * cols
			if size != 4 && size != 6 && size != 9 {
				return 0, 0, errors.New("grid size must be 4, 6, or 9")
			}
			return rows, cols, nil
		}
	}

	if size, err := strconv.Atoi(value); err == nil {
		switch size {
		case 4:
			return 2, 2, nil
		case 6:
			return 2, 3, nil
		case 9:
			return 3, 3, nil
		}
	}

	return 0, 0, errors.New("grid size must be 4, 6, or 9 (or 2x2/2x3/3x3)")
}

// Init initializes the application.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.loadProjects(),
		a.loadProfiles(),
	)
}

// loadProjects returns a command to load projects.
func (a App) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := a.store.List(a.ctx)
		return ProjectsLoadedMsg{Projects: projects, Err: err}
	}
}

// loadProfiles returns a command to load profiles.
func (a App) loadProfiles() tea.Cmd {
	return func() tea.Msg {
		profiles, err := a.store.ListProfiles(a.ctx)
		return ProfilesLoadedMsg{Profiles: profiles, Err: err}
	}
}

func (a *App) updateAddDialogProfiles() {
	if len(a.profiles) == 0 {
		a.addDialog.SetFieldOptions(2, nil)
		return
	}
	options := make([]string, 0, len(a.profiles))
	for _, p := range a.profiles {
		label := p.Name
		if p.IsDefault {
			label = p.Name + " (default)"
		}
		options = append(options, label+" ("+p.ID+")")
	}
	a.addDialog.SetFieldOptions(2, options)
}

// startSession starts a PTY session for the selected project.
func (a *App) startSession(project *model.Project) tea.Cmd {
	return func() tea.Msg {
		// Get profile for project
		profile, err := a.store.GetProfile(a.ctx, project.ProfileID)
		if err != nil {
			// Use default profile
			profile, _ = a.store.GetDefault(a.ctx)
		}

		// Create session
        // Get initial dimensions from the terminal instance if it exists
        rows := 24
        cols := 80
        if inst, ok := a.terminals[project.ID]; ok {
            w, h := inst.Terminal.PTYSize()
            if w > 0 && h > 0 {
                cols = w
                rows = h
            }
        }
		_, err = a.engine.CreateSession(a.ctx, project, profile, rows, cols)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return SessionStartedMsg{ProjectID: project.ID}
	}
}

// waitForOutput waits for session output.
func (a *App) waitForOutput(projectID string) tea.Cmd {
	session, ok := a.engine.GetSession(projectID)
	if !ok {
		return nil
	}
	return WaitForOutput(session.Output(), projectID)
}

// SetSize updates the window dimensions.
func (a *App) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.ready = true

	if a.windowTooSmall() {
		a.statusBar.SetWidth(width)
		a.addDialog.SetSize(width, height)
		a.profileDialog.SetSize(width, height)
		a.settingsDialog.SetSize(width, height)
		a.commandDialog.SetSize(width, height)
		return
	}

	leftWidth, rightWidth, contentHeight, colWidths, rowHeights := a.gridLayout()
	_, cols := a.gridActiveDims()

	// Set component sizes
	a.projectList.SetSize(leftWidth, contentHeight)
	a.sessionTabs.SetWidth(rightWidth)
	a.statusBar.SetWidth(width)

	// Set terminal sizes per grid cell
	ids := a.gridOrder()
	for i, id := range ids {
		inst, ok := a.terminals[id]
		if !ok {
			continue
		}
		row := 0
		col := 0
		if cols > 0 {
			row = i / cols
			col = i % cols
		}
		if row >= len(rowHeights) || col >= len(colWidths) {
			continue
		}
		inst.Terminal.SetSize(colWidths[col], rowHeights[row])
		if session, ok := a.engine.GetSession(id); ok && session.Status() == model.SessionStatusRunning {
			cols, rows := inst.Terminal.PTYSize()
			// Enforce minimum PTY size to prevent CLI tool crashes/OOM
			if cols < 8 {
				cols = 8
			}
			if rows < 2 {
				rows = 2
			}
			if cols > 0 && rows > 0 {
				_ = session.Resize(uint16(rows), uint16(cols))
			}
		}
	}

	// Dialog size
	a.addDialog.SetSize(width, height)
	a.profileDialog.SetSize(width, height)
	a.settingsDialog.SetSize(width, height)
	a.commandDialog.SetSize(width, height)

	// Profile manager size
	pmWidth, pmHeight := a.profileManagerSize()
	a.profileList.SetSize(pmWidth, pmHeight)

	// Update focus styles
	a.updateFocusStyles()
}

func (a App) windowTooSmall() bool {
	return a.width < minAppWidth || a.height < minAppHeight
}

func (a *App) closeSession(projectID string) {
	if projectID == "" {
		return
	}
	_ = a.engine.CloseSession(projectID)
	a.projectList.SetRunning(projectID, false)
	a.sessionTabs.RemoveTab(projectID)
	delete(a.terminals, projectID)
	delete(a.outputWatchers, projectID)
	a.normalizeActivePane()
	a.SetSize(a.width, a.height)
}

func (a *App) profileManagerSize() (int, int) {
	width := a.width * 70 / 100
	height := a.height * 70 / 100
	if width < 50 {
		width = 50
	}
	if height < 16 {
		height = 16
	}
	if width > a.width-4 {
		width = a.width - 4
	}
	if height > a.height-4 {
		height = a.height - 4
	}
	return width, height
}

func (a *App) gridCapacity() int {
	if a.gridRows < 1 || a.gridCols < 1 {
		return 0
	}
	return a.gridRows * a.gridCols
}

func (a *App) gridActiveDims() (int, int) {
	return gridDimsForCount(len(a.sessionTabs.Tabs()), a.gridRows, a.gridCols)
}

func gridDimsForCount(count, maxRows, maxCols int) (int, int) {
	if count <= 0 || maxRows < 1 || maxCols < 1 {
		return 0, 0
	}

	rows := 1
	cols := 1

	for rows*cols < count {
		if cols < maxCols && (cols <= rows || rows == maxRows) {
			cols++
			continue
		}
		if rows < maxRows {
			rows++
			continue
		}
		if cols < maxCols {
			cols++
			continue
		}
		break
	}

	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	if rows > maxRows {
		rows = maxRows
	}
	if cols > maxCols {
		cols = maxCols
	}

	return rows, cols
}

func (a *App) gridOrder() []string {
	capacity := a.gridCapacity()
	if capacity == 0 {
		return nil
	}
	tabs := a.sessionTabs.Tabs()
	if len(tabs) == 0 {
		return nil
	}
	if len(tabs) > capacity {
		tabs = tabs[:capacity]
	}
	ids := make([]string, 0, len(tabs))
	for _, t := range tabs {
		ids = append(ids, t.ID)
	}
	return ids
}

func (a *App) setActivePane(index int) {
	ids := a.gridOrder()
	if len(ids) == 0 {
		a.activePane = 0
		a.activeTermID = ""
		a.updateFocusStyles()
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(ids) {
		index = len(ids) - 1
	}
	a.activePane = index
	a.activeTermID = ids[index]
	a.sessionTabs.SetActiveTab(ids[index])
	a.updateFocusStyles()
}

func (a *App) setActivePaneByProject(projectID string) {
	ids := a.gridOrder()
	for i, id := range ids {
		if id == projectID {
			a.setActivePane(i)
			return
		}
	}
	a.activeTermID = projectID
	a.updateFocusStyles()
}

func (a *App) normalizeActivePane() {
	ids := a.gridOrder()
	if len(ids) == 0 {
		a.activePane = 0
		a.activeTermID = ""
		a.updateFocusStyles()
		return
	}
	if a.activePane >= len(ids) {
		a.activePane = len(ids) - 1
	}
	a.activeTermID = ids[a.activePane]
	a.sessionTabs.SetActiveTab(a.activeTermID)
	a.updateFocusStyles()
}

func (a *App) gridLayout() (int, int, int, []int, []int) {
	// Left panel (project list): 25% width
	leftWidth := a.width * 25 / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	if leftWidth > 40 {
		leftWidth = 40
	}
	rightWidth := a.width - leftWidth
	contentHeight := a.height - 1

	rows, cols := a.gridActiveDims()
	colWidths := distribute(rightWidth, cols)
	rowHeights := distribute(contentHeight, rows)

	return leftWidth, rightWidth, contentHeight, colWidths, rowHeights
}

func distribute(total, parts int) []int {
	if parts <= 0 {
		return nil
	}
	if total < parts {
		parts = total
	}
	base := 0
	rem := 0
	if parts > 0 {
		base = total / parts
		rem = total % parts
	}
	out := make([]int, parts)
	for i := 0; i < parts; i++ {
		out[i] = base
		if i < rem {
			out[i]++
		}
		if out[i] < 1 {
			out[i] = 1
		}
	}
	return out
}

func (a *App) hasPane(projectID string) bool {
	for _, t := range a.sessionTabs.Tabs() {
		if t.ID == projectID {
			return true
		}
	}
	return false
}

func (a *App) canOpenPane(projectID string) bool {
	if a.hasPane(projectID) {
		return true
	}
	capacity := a.gridCapacity()
	if capacity == 0 {
		return false
	}
	return len(a.sessionTabs.Tabs()) < capacity
}

// updateFocusStyles updates component focus states.
func (a *App) updateFocusStyles() {
	a.projectList.SetFocused(a.focus == FocusProjects)
	for id, inst := range a.terminals {
		isFocused := false
		if a.focus == FocusTerminal {
			if a.dispatchMode == DispatchModeBroadcast || a.dispatchMode == DispatchModeChain {
				isFocused = true
			} else {
				isFocused = id == a.activeTermID
			}
		}
		inst.Terminal.SetFocused(isFocused)
	}
	a.statusBar.SetModeLabel(a.inputModeLabel())
}

// cycleFocus switches to the next focus area.
func (a *App) cycleFocus() {
	ids := a.gridOrder()
	if len(ids) == 0 {
		a.focus = FocusProjects
		a.updateFocusStyles()
		return
	}

	switch a.focus {
	case FocusProjects:
		a.focus = FocusTerminal
		if a.activeTermID != "" {
			a.setActivePaneByProject(a.activeTermID)
		} else {
			a.setActivePane(0)
		}
	case FocusTerminal:
		index := indexOfID(ids, a.activeTermID)
		if index == -1 {
			index = 0
		}
		if index+1 < len(ids) {
			a.setActivePane(index + 1)
			a.focus = FocusTerminal
		} else {
			a.focus = FocusProjects
		}
	}
	a.updateFocusStyles()
}

func (a *App) cycleFocusReverse() {
	ids := a.gridOrder()
	if len(ids) == 0 {
		a.focus = FocusProjects
		a.updateFocusStyles()
		return
	}

	switch a.focus {
	case FocusTerminal:
		index := indexOfID(ids, a.activeTermID)
		if index == -1 {
			index = 0
		}
		if index-1 >= 0 {
			a.setActivePane(index - 1)
			a.focus = FocusTerminal
		} else {
			a.focus = FocusProjects
		}
	case FocusProjects:
		a.focus = FocusTerminal
		a.setActivePane(len(ids) - 1)
	}
	a.updateFocusStyles()
}

func (a *App) inputModeLabel() string {
	var inputLabel string
	if a.inputMode == InputModeTerminal {
		inputLabel = "TERM"
	} else {
		inputLabel = "CTRL"
	}

	var dispatchLabel string
	switch a.dispatchMode {
	case DispatchModeBroadcast:
		dispatchLabel = "BCAST"
	case DispatchModeChain:
		dispatchLabel = "CHAIN"
	default:
		dispatchLabel = "SOLO"
	}

	return inputLabel + "|" + dispatchLabel
}

func (a *App) enterTerminalMode() {
	if a.inputMode == InputModeTerminal {
		return
	}
	if a.activeTermID == "" {
		return
	}
	a.inputMode = InputModeTerminal
	a.focus = FocusTerminal
	a.updateFocusStyles()
}

func (a *App) enterControlMode() {
	if a.inputMode == InputModeControl {
		return
	}
	a.inputMode = InputModeControl
	a.updateFocusStyles()
}



func (a *App) toggleInputMode() {
	if a.inputMode == InputModeTerminal {
		a.enterControlMode()
		return
	}
	a.enterTerminalMode()
}

func indexOfID(ids []string, id string) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}

// getOrCreateTerminal gets or creates a terminal instance for a project.
func (a *App) getOrCreateTerminal(projectID, projectName string) *TerminalInstance {
	if inst, ok := a.terminals[projectID]; ok {
		return inst
	}

	// Create new terminal instance
	term := terminal.New()
	term.SetProject(projectID, projectName)

	_, _, _, colWidths, rowHeights := a.gridLayout()
	cellWidth := 0
	cellHeight := 0
	if len(colWidths) > 0 {
		cellWidth = colWidths[0]
	}
	if len(rowHeights) > 0 {
		cellHeight = rowHeights[0]
	}
	if cellWidth < 1 {
		cellWidth = 40
	}
	if cellHeight < 1 {
		cellHeight = 10
	}
	term.SetSize(cellWidth, cellHeight)

	inst := &TerminalInstance{
		ProjectID:   projectID,
		ProjectName: projectName,
		Terminal:    term,
	}
	a.terminals[projectID] = inst

	return inst
}

// showAddDialog shows the add project dialog.
func (a *App) showAddDialog() {
	a.dialogMode = DialogAddProject
	a.addDialog.Reset()
}

func (a *App) showProfileManager() {
	a.dialogMode = DialogManageProfiles
	a.profileList.SetProfiles(a.profiles)
	a.profileList.SetFocused(true)
}

func (a *App) showSettingsDialog() {
	rows := strconv.Itoa(a.gridRows)
	cols := strconv.Itoa(a.gridCols)
	
	a.settingsDialog = dialog.NewInputDialog("Settings", []dialog.InputField{
		{Label: "Grid Size (e.g. 2x2, 3x3, 4, 6)", Placeholder: "2x2", Value: rows+"x"+cols},
	})
	a.settingsDialog.SetSize(a.width, a.height)
	a.dialogMode = DialogSettings
}


func (a *App) showFilePreview() {
	if a.turnFilename == "" {
		a.statusBar.SetMessage("No active organizer file to preview", true)
		return
	}
	
	a.filePreview.SetFile(a.turnFilename)
	a.filePreview.SetSize(a.width, a.height)
	a.dialogMode = DialogFilePreview
}


func (a *App) showCommandDialog() {
	a.commandDialog = dialog.NewInputDialog("Command", []dialog.InputField{
		{Label: "Command", Placeholder: "quit"},
	})
	a.commandDialog.SetSize(a.width, a.height)
	a.dialogMode = DialogCommand
}

func (a *App) showProfileDialog(profile *model.Profile) {
	a.profileEditID = ""
	title := "Add Profile"
	if profile != nil {
		a.profileEditID = profile.ID
		title = "Edit Profile"
	}

	commandValue := ""
	envValue := ""
	nameValue := ""
	if profile != nil {
		nameValue = profile.Name
		commandValue = strings.TrimSpace(profile.Command)
		envValue = utils.FormatEnvVars(profile.EnvVars)
	}

	a.profileDialog = dialog.NewInputDialog(title, []dialog.InputField{
		{Label: "Profile Name", Placeholder: "My Profile", Value: nameValue},
		{Label: "Command", Placeholder: "claude, codex, or ccr code", Value: commandValue},
		{Label: "Env Vars", Placeholder: "KEY=VALUE, KEY2=VALUE2", Value: envValue},
	})
	a.profileDialog.SetSize(a.width, a.height)
	a.dialogMode = DialogEditProfile
}

// hideDialog hides any open dialog.
func (a *App) hideDialog() {
	a.dialogMode = DialogNone
	a.profileList.SetFocused(false)
}

func (a *App) updateGridSettings(rows, cols int) error {
	if rows < 1 || rows > 3 || cols < 1 || cols > 3 {
		return errors.New("grid rows/cols must be between 1 and 3")
	}
	size := rows * cols
	if size != 4 && size != 6 && size != 9 {
		return errors.New("grid size must be 4, 6, or 9")
	}

	if a.config != nil && a.configDir != "" {
		updated := *a.config
		updated.GridRows = rows
		updated.GridCols = cols
		if err := app.SaveConfig(a.configDir, &updated); err != nil {
			return err
		}
		*a.config = updated
	}

	a.gridRows = rows
	a.gridCols = cols
	a.normalizeActivePane()
	a.SetSize(a.width, a.height)
	return nil
}

func (a *App) executeCommand(input string) tea.Cmd {
	cmd := strings.TrimSpace(input)
	if cmd == "" {
		return nil
	}
	if strings.HasPrefix(cmd, ":") {
		cmd = strings.TrimSpace(strings.TrimPrefix(cmd, ":"))
	}
	switch strings.ToLower(cmd) {
	case "q", "wq", "quit", "exit":
		a.quitting = true
		a.engine.CloseAll()
		return tea.Quit
	default:
		a.statusBar.SetMessage("Unknown command: "+cmd, true)
		return nil
	}
}

// createProject creates a new project from dialog values.
func (a *App) createProject() tea.Cmd {
	values := a.addDialog.Values()
	name := strings.TrimSpace(values[0])
	path := strings.TrimSpace(values[1])
	profileInput := ""
	if len(values) > 2 {
		profileInput = strings.TrimSpace(values[2])
	}

	if name == "" || path == "" {
		a.statusBar.SetMessage("Name and path are required", true)
		return nil
	}

	// Expand and validate path
	path = utils.ExpandPath(path)
	if !utils.IsValidProjectPath(path) {
		a.statusBar.SetMessage("Invalid project path: directory does not exist", true)
		return nil
	}

	project := model.NewProject(name, path)
	if profileInput != "" {
		profileID, err := a.resolveProfileID(profileInput)
		if err != nil {
			a.statusBar.SetMessage(err.Error(), true)
			return nil
		}
		project.ProfileID = profileID
	}

	return func() tea.Msg {
		if err := a.store.Create(a.ctx, project); err != nil {
			return ErrorMsg{Err: err}
		}
		return ProjectCreatedMsg{Project: *project}
	}
}

func (a *App) buildProfileFromDialog() (*model.Profile, bool, error) {
	values := a.profileDialog.Values()
	if len(values) < 3 {
		return nil, false, errors.New("profile form is incomplete")
	}

	name := strings.TrimSpace(values[0])
	command := strings.TrimSpace(values[1])
	envInput := strings.TrimSpace(values[2])

	if name == "" {
		return nil, false, errors.New("profile name is required")
	}

	var existing *model.Profile
	if a.profileEditID != "" {
		existing = a.findProfileByID(a.profileEditID)
		if existing == nil {
			return nil, false, errors.New("profile not found")
		}
	}

	if command == "" {
		if existing != nil && existing.Command != "" {
			command = existing.Command
		} else {
			command = defaultProfileCommand()
		}
	}

	envVars, err := utils.ParseEnvVars(envInput)
	if err != nil {
		return nil, false, err
	}

	if existing != nil {
		updated := *existing
		updated.Name = name
		updated.Command = command
		updated.EnvVars = envVars
		updated.Driver = model.DriverNative
		updated.CommandArgs = nil
		return &updated, false, nil
	}

	profile := model.NewProfile(name)
	profile.Command = command
	profile.EnvVars = envVars
	profile.Driver = model.DriverNative
	profile.CommandArgs = nil
	return profile, true, nil
}

func defaultProfileCommand() string {
	return "claude"
}

func (a *App) saveProfile(profile *model.Profile, isNew bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if isNew {
			err = a.store.CreateProfile(a.ctx, profile)
		} else {
			err = a.store.UpdateProfile(a.ctx, profile)
		}
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ProfileSavedMsg{Profile: *profile, IsNew: isNew}
	}
}

func (a *App) deleteProfile(id string) tea.Cmd {
	return func() tea.Msg {
		if err := a.store.DeleteProfile(a.ctx, id); err != nil {
			return ErrorMsg{Err: err}
		}
		return ProfileDeletedMsg{ProfileID: id}
	}
}

func (a *App) upsertProfileInMemory(profile model.Profile) {
	updated := false
	for i := range a.profiles {
		if a.profiles[i].ID == profile.ID {
			a.profiles[i] = profile
			updated = true
			break
		}
	}
	if !updated {
		a.profiles = append(a.profiles, profile)
	}
	a.updateAddDialogProfiles()
	a.profileList.SetProfiles(a.profiles)
}

func (a *App) setDefaultProfile(id string) tea.Cmd {
	return func() tea.Msg {
		profiles, err := a.store.ListProfiles(a.ctx)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		changed := false
		for i := range profiles {
			shouldBeDefault := profiles[i].ID == id
			if profiles[i].IsDefault != shouldBeDefault {
				profiles[i].IsDefault = shouldBeDefault
				if err := a.store.UpdateProfile(a.ctx, &profiles[i]); err != nil {
					return ErrorMsg{Err: err}
				}
				changed = true
			}
		}
		if !changed {
			return ProfilesLoadedMsg{Profiles: profiles}
		}
		updated, err := a.store.ListProfiles(a.ctx)
		return ProfilesLoadedMsg{Profiles: updated, Err: err}
	}
}

func (a *App) findProfileByID(id string) *model.Profile {
	for i := range a.profiles {
		if a.profiles[i].ID == id {
			return &a.profiles[i]
		}
	}
	return nil
}

func (a *App) findProjectByID(id string) *model.Project {
	for i := range a.projects {
		if a.projects[i].ID == id {
			return &a.projects[i]
		}
	}
	return nil
}

func (a *App) profileForProject(project *model.Project) *model.Profile {
	if project == nil {
		return nil
	}
	if project.ProfileID != "" {
		if profile := a.findProfileByID(project.ProfileID); profile != nil {
			return profile
		}
	}
	for i := range a.profiles {
		if a.profiles[i].IsDefault {
			return &a.profiles[i]
		}
	}
	if len(a.profiles) > 0 {
		return &a.profiles[0]
	}
	return nil
}

func (a *App) dispatchNotifications(profile *model.Profile, events []notify.Event) tea.Cmd {
	if a.notifier == nil || len(events) == 0 {
		return nil
	}
	cfg := model.NotificationConfig{Desktop: true}
	if profile != nil {
		cfg = profile.Notification
	}
	return func() tea.Msg {
		for _, ev := range events {
			a.notifier.Dispatch(a.ctx, cfg, ev)
		}
		return nil
	}
}

func (a *App) resolveProfileID(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", nil
	}

	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)

	// Try parse trailing "(id)"
	if idx := strings.LastIndex(trimmed, "("); idx != -1 && strings.HasSuffix(trimmed, ")") && idx < len(trimmed)-1 {
		candidate := strings.TrimSpace(trimmed[idx+1 : len(trimmed)-1])
		for _, p := range a.profiles {
			if p.ID == candidate {
				return p.ID, nil
			}
		}
	}

	// Exact ID match
	for _, p := range a.profiles {
		if p.ID == trimmed {
			return p.ID, nil
		}
	}

	// "default" keyword
	if lower == "default" {
		for _, p := range a.profiles {
			if p.IsDefault {
				return p.ID, nil
			}
		}
	}

	// Name match (case-insensitive)
	var match *model.Profile
	for i := range a.profiles {
		if strings.EqualFold(a.profiles[i].Name, trimmed) {
			if match != nil {
				return "", errors.New("multiple profiles share the same name")
			}
			match = &a.profiles[i]
		}
	}
	if match != nil {
		return match.ID, nil
	}

	return "", errors.New("unknown profile: " + trimmed)
}
