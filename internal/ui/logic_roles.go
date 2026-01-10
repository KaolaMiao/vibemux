package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazyvibe/vibemux/internal/model"
	"github.com/lazyvibe/vibemux/internal/ui/components/dialog"
)

// Role Prompts (Default presets)
const (
	// Suffix to instruct brief confirmation
	RolePromptConfirmation = " [系统指令：请确认你的角色。无需多言，仅回复“收到”即可。]"

	RolePromptJudge = `你现在是【裁判长/调度员】。
任务目标：[在此填入待讨论的议题]
职责：不参与辩论，只负责分析【正方】和【反方】的论据。
` + RolePromptConfirmation

	RolePromptProponent = `你现在是【正方】。
职责：坚定支持该议题。提供具体的论据。
` + RolePromptConfirmation

	RolePromptOpponent = `你现在是【反方】。
职责：对该议题持有审慎或反对态度。寻找漏洞。
` + RolePromptConfirmation

	RolePromptObserver = `你现在是【观察员】。
职责：记录会议要点，不直接参与讨论。
` + RolePromptConfirmation
)

// showRoleDialog opens the dialog to assign roles to active terminals.
func (a *App) showRoleDialog() {
	ids := a.gridOrder()
	if len(ids) == 0 {
		a.statusBar.SetMessage("No active terminals to assign roles", true)
		return
	}

	var fields []dialog.InputField

	for i, id := range ids {
		inst, ok := a.terminals[id]
		if !ok {
			continue
		}

		// Determine default role based on index
		label := fmt.Sprintf("%s (%s)", inst.ProjectName, id)
		defaultPrompt := ""
		roleName := "Observer"

		switch i {
		case 0:
			roleName = "JUDGE (A)"
			defaultPrompt = RolePromptJudge
		case 1:
			roleName = "PROPONENT (B)"
			defaultPrompt = RolePromptProponent
		case 2:
			roleName = "OPPONENT (C)"
			defaultPrompt = RolePromptOpponent
		default:
			roleName = fmt.Sprintf("OBSERVER (%d)", i)
			defaultPrompt = RolePromptObserver
		}

		fields = append(fields, dialog.InputField{
			Label:       fmt.Sprintf("[%s] %s", roleName, label),
			Placeholder: "Enter system prompt for this agent...",
			Value:       defaultPrompt,
		})
	}

	a.roleDialog = dialog.NewInputDialog("Assign System Roles", fields)
	a.roleDialog.SetSize(a.width, a.height)
	a.dialogMode = DialogAssignRoles
}

// assignRolesToTerminals sends the entered prompts to the respective terminals.
func (a *App) assignRolesToTerminals() []tea.Cmd {
	ids := a.gridOrder()
	values := a.roleDialog.Values()
	var cmds []tea.Cmd

	for i, id := range ids {
		if i >= len(values) {
			break
		}
		
		prompt := values[i]
		if prompt == "" {
			continue
		}

		// Construct the injection command
		projectID := id
		promptText := prompt

		cmds = append(cmds, func() tea.Msg {
			session, ok := a.engine.GetSession(projectID)
			if !ok || session.Status() != model.SessionStatusRunning {
				return nil
			}

			// Format the injection
			// We add a few newlines to ensure it stands out
			// Note: We deliberately do NOT use the special :::VIBE_OUTPUT::: delimiters here
			// because this is a system instruction, not a chain context injection.
			injection := fmt.Sprintf("\n\n%s", promptText)
			
			session.Write([]byte(injection))
			
			// Auto-submit with a slight delay to ensure paste is processed
			time.Sleep(200 * time.Millisecond)
			session.Write([]byte("\r"))
			
			return nil
		})
	}

	return cmds
}
