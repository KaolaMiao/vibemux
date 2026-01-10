package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/ui/components/configdialog"
)

// File-Based Role Prompts (Organizer Mode)
const (
	RolePromptFileOrganizer = `
[SYSTEM INSTRUCTION - 组织者角色]
你是本次协作会议的【组织者】(Organizer)。
主题：{{TOPIC}}
目标文件：{{FILENAME}}

[文件操作规范]
- **创建方式**：使用你的文件工具创建文件，或使用 shell 命令
- **绝对路径**：始终使用完整路径 {{FILENAME}}
- **授权**：你已被授权读写 {{FILENAME}} 及其所在目录

[任务清单]
1. **创建文件**：创建 "{{FILENAME}}"，写入以下内容：
   - 标题：# {{TOPIC}}
   - 时间：会议开始时间
   - 参与者区域：为每个角色预留发言区
   - 规则说明：发言格式为 ` + "`### [角色名] (时间戳)`" + `

2. **保持中立**：你绝对**不能**发表任何关于议题的个人观点。

3. **确认完成**：创建完成后，回复 "文件已创建，结构已就绪"。
`

	RolePromptFileParticipant = `
[SYSTEM INSTRUCTION - 参与者角色]
你是本次协作会议的【{{ROLE}}】。
主题：{{TOPIC}}
目标文件：{{FILENAME}}

[文件操作规范]
- **读取方式**：使用文件工具读取，或 cat "{{FILENAME}}"
- **追加方式**：使用文件追加功能，或 echo "内容" >> "{{FILENAME}}"
- **禁止覆盖**：绝对不能使用 > 覆盖文件，只能追加 >>
- **授权**：你已被授权读写 {{FILENAME}}

[初始化阶段 - 立即执行]
收到此消息后，仅回复 "Role Received: {{ROLE}}" 并等待。
- 不要读取文件
- 不要输出观点
- 不要使用任何工具

[工作阶段 - 收到 "[SYSTEM] Your Turn" 后执行]
1. **读取**：先读取 {{FILENAME}} 的全部内容，了解当前进展
2. **思考**：基于你的角色 {{ROLE}}，思考你的观点
3. **追加**：将你的观点**追加**到文件末尾，格式为：
   ` + "`### [{{ROLE}}] (当前时间)`" + `
   你的观点内容...

4. **确认**：写入完成后，回复 "观点已写入"
`

	RolePromptFileCommon = `
[SYSTEM INSTRUCTION]
当前会话采用【共享文件协同模式】。
历史记录文件路径： %s
请务必遵守以下规则：
1. **读取**：每次回答前，必须先读取该文件的全部内容，了解辩论进展。
2. **写入**：你的发言必须**追加写入**到该文件的末尾。使用 Markdown 格式。
3. **格式**：每一段发言必须以 ` + "`### [你的角色名] (这里填时间)`" + ` 开头。
4. **人设**：严禁偏离你的设定角色，言行必须符合【%s】的身份。
5. **静默**：屏幕上不要输出大段辩论内容，仅输出 "已写入观点到日志" 即可。
`

	RolePromptFileJudge = `你现在是【裁判长/调度员】。
任务目标：[在此填入议题]
职责：监控辩论进程，在文件中分析局势。`

	RolePromptFileProponent = `你现在是【正方】。
职责：支持议题，将你的论据写入文件。`

	RolePromptFileOpponent = `你现在是【反方】。
职责：反对议题，将你的质疑写入文件。`

	RolePromptFileObserver = `你现在是【观察员】。
职责：在文件中记录要点。`
)

// showRoleDialogFile opens the file-based role assignment dialog.
func (a *App) showRoleDialogFile() {
	ids := a.gridOrder()
	if len(ids) == 0 {
		a.statusBar.SetMessage("No active terminals", true)
		return
	}

	var fields []configdialog.Field

	// --- Left Column: Context ---

	// Field 0: Topic
	fields = append(fields, configdialog.Field{
		Label:       "Meeting Topic",
		Placeholder: "e.g. AI_Safety",
		Value:       "Project_Discussion",
		Type:        configdialog.InputText,
		Column:      0,
		Header:      "[Context]",
	})

	// Field 1: Filename
	fields = append(fields, configdialog.Field{
		Label:       "Log Filename",
		Placeholder: ".vibemux/log.md",
		Value:       ".vibemux/discussion.md",
		Type:        configdialog.InputText,
		Column:      0,
	})

	// Field 2: Turn Sequence
	defaultSeq := ""
	for i := range ids {
		if i > 0 {
			defaultSeq += ","
		}
		defaultSeq += fmt.Sprintf("%d", i)
	}
	fields = append(fields, configdialog.Field{
		Label:       "Turn Sequence",
		Placeholder: "0,1,2",
		Value:       defaultSeq,
		Type:        configdialog.InputText,
		Column:      0,
	})

	// --- Right Column: Terminals ---
	
	// Get grid dimensions to calculate positions
	_, cols := a.gridActiveDims()
	if cols == 0 { cols = 1 } // Prevent division by zero

	for i, id := range ids {
		inst, _ := a.terminals[id]
		
		// Header for Terminal
		termHeader := fmt.Sprintf("[%d] %s", i, inst.ProjectName)
		
		// Calculate Grid Position
		row := i / cols
		col := i % cols
		
		// Defaults
		var roleDefault, promptDefault string
		if i == 0 {
			roleDefault = "ORGANIZER"
			promptDefault = RolePromptFileOrganizer
		} else {
			if i == 1 { roleDefault = "PROPONENT" } else 
			if i == 2 { roleDefault = "OPPONENT" } else 
			{ roleDefault = fmt.Sprintf("OBSERVER_%d", i) }
			
			promptDefault = RolePromptFileParticipant
		}

		// Field: Role Name (Text)
		fields = append(fields, configdialog.Field{
			Label:       "Role Name",
			Value:       roleDefault,
			Type:        configdialog.InputText,
			Column:      1,
			Header:      termHeader,
			GridRow:     row,
			GridCol:     col,
		})
		
		// Field: Prompt (TextArea)
		fields = append(fields, configdialog.Field{
			Label:       "System Prompt",
			Value:       promptDefault,
			Type:        configdialog.InputTextArea,
			Column:      1,
			GridRow:     row,
			GridCol:     col,
		})
	}

	a.organizerDialog = configdialog.New("Assign Roles (Organizer Mode)", fields)
	a.organizerDialog.SetSize(a.width, a.height)
	a.dialogMode = DialogAssignRolesFile
}


// assignRolesToTerminalsFile handles the submission of file-based roles.
func (a *App) assignRolesToTerminalsFile() []tea.Cmd {
	ids := a.gridOrder()
	values := a.organizerDialog.Values()
	var cmds []tea.Cmd
	
	// Expected fields: 
	// 0: Topic
	// 1: Filename
	// 2: Sequence
	// Then 2 fields per terminal: Role, Prompt.
	
	if len(values) < 3 + len(ids)*2 {
		a.statusBar.SetMessage("Error: Missing fields", true)
		return nil
	}

	// 1. Extract Global Config
	topic := strings.TrimSpace(values[0])
	if topic == "" { topic = "Project_Discussion" }
	
	filename := strings.TrimSpace(values[1])
	if filename == "" {
		filenameBase := strings.ReplaceAll(topic, " ", "_")
		filename = fmt.Sprintf(".vibemux/%s.md", filenameBase)
	}
	
	// Compute ABSOLUTE path based on first project's directory
	// so FilePreview can read from the correct location
	// AI agents run in the project's working directory, so they will create files there
	var basePath string
	if len(ids) > 0 {
		if proj := a.findProjectByID(ids[0]); proj != nil && proj.Path != "" {
			basePath = proj.Path
		}
	}
	// Fallback: if no project path, try to use first terminal's known path
	if basePath == "" {
		for id := range a.terminals {
			if proj := a.findProjectByID(id); proj != nil && proj.Path != "" {
				basePath = proj.Path
				break
			}
		}
	}
	
	if basePath != "" {
		absFilename := filepath.Join(basePath, filename)
		_ = os.MkdirAll(filepath.Dir(absFilename), 0755)
		filename = absFilename
	}

	seqStr := strings.TrimSpace(values[2])
	a.turnTopic = topic
	a.turnFilename = filename
	
	// Initialize Auto-Turn (Paused)
	a.initAutoTurn(seqStr)

	// 2. Process Terminals
	baseIdx := 3
	for i, id := range ids {
		projectID := id
		
		// Extract Role & Prompt
		// i=0 -> baseIdx + 0, baseIdx + 1
		// i=1 -> baseIdx + 2, baseIdx + 3
		roleIdx := baseIdx + (i * 2)
		promptIdx := roleIdx + 1
		
		roleName := strings.TrimSpace(values[roleIdx])
		rawPrompt := values[promptIdx]
		
		// Template Replacement
		finalPrompt := strings.ReplaceAll(rawPrompt, "{{TOPIC}}", topic)
		finalPrompt = strings.ReplaceAll(finalPrompt, "{{FILENAME}}", filename)
		finalPrompt = strings.ReplaceAll(finalPrompt, "{{ROLE}}", roleName)

		cmds = append(cmds, func() tea.Msg {
			session, ok := a.engine.GetSession(projectID)
			if !ok || session.Status() != model.SessionStatusRunning {
				return nil
			}

			// Pre-fill prompt ONLY (No execution)
			// We add newlines to ensure clean separation from previous output
			injection := fmt.Sprintf("\n\n%s", finalPrompt)
			session.Write([]byte(injection))
			// REMOVED: Automatic carriage return (\r)
			return nil
		})
	}

	return cmds
}
