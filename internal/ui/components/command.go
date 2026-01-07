package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/ui/theme"
)

// Command represents a available command
type Command struct {
	Name        string
	Aliases     []string
	Description string
}

// AvailableCommands lists all supported commands
var AvailableCommands = []Command{
	{Name: "stacks", Aliases: []string{"st", "stack", "cfn"}, Description: "CloudFormation stacks"},
	{Name: "services", Aliases: []string{"svc", "service", "ecs"}, Description: "ECS services"},
	{Name: "clusters", Aliases: []string{"cl", "cluster"}, Description: "ECS clusters"},
	{Name: "tasks", Aliases: []string{"task", "t"}, Description: "ECS tasks"},
	{Name: "tunnels", Aliases: []string{"tun", "tunnel", "pf"}, Description: "Port forward tunnels"},
	{Name: "lambda", Aliases: []string{"fn", "functions", "lam"}, Description: "Lambda functions"},
	{Name: "apigateway", Aliases: []string{"apigw", "api", "gw"}, Description: "API Gateway"},
	{Name: "logs", Aliases: []string{"log", "l"}, Description: "Toggle logs panel"},
	{Name: "refresh", Aliases: []string{"r", "reload"}, Description: "Refresh current view"},
	{Name: "quit", Aliases: []string{"q", "exit"}, Description: "Quit application"},
	{Name: "help", Aliases: []string{"h", "?"}, Description: "Show help"},
}

// CommandResult is the result of executing a command
type CommandResult struct {
	Command string
	Args    []string
}

// CommandPalette provides k9s-style command input
type CommandPalette struct {
	input       textinput.Model
	active      bool
	width       int
	suggestions []Command
}

// NewCommandPalette creates a new command palette
func NewCommandPalette() *CommandPalette {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.CharLimit = 64
	ti.Width = 30

	return &CommandPalette{
		input:       ti,
		active:      false,
		suggestions: []Command{},
	}
}

// SetWidth sets the palette width
func (c *CommandPalette) SetWidth(width int) {
	c.width = width
	c.input.Width = min(50, width-10)
}

// Activate shows the command palette
func (c *CommandPalette) Activate() tea.Cmd {
	c.active = true
	c.input.SetValue("")
	c.input.Focus()
	c.updateSuggestions()
	return textinput.Blink
}

// Deactivate hides the command palette
func (c *CommandPalette) Deactivate() {
	c.active = false
	c.input.Blur()
	c.input.SetValue("")
	c.suggestions = []Command{}
}

// IsActive returns whether the palette is active
func (c *CommandPalette) IsActive() bool {
	return c.active
}

// Update handles input updates
func (c *CommandPalette) Update(msg tea.Msg) (*CommandResult, tea.Cmd) {
	if !c.active {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Execute command
			result := c.parseCommand()
			c.Deactivate()
			return result, nil

		case "esc":
			c.Deactivate()
			return nil, nil

		case "tab":
			// Auto-complete first suggestion
			if len(c.suggestions) > 0 {
				c.input.SetValue(c.suggestions[0].Name)
				c.input.CursorEnd()
				c.updateSuggestions()
			}
			return nil, nil
		}
	}

	// Update text input
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	c.updateSuggestions()

	return nil, cmd
}

// updateSuggestions updates command suggestions based on input
func (c *CommandPalette) updateSuggestions() {
	query := strings.ToLower(strings.TrimSpace(c.input.Value()))
	c.suggestions = []Command{}

	if query == "" {
		c.suggestions = AvailableCommands
		return
	}

	for _, cmd := range AvailableCommands {
		// Check name
		if strings.HasPrefix(strings.ToLower(cmd.Name), query) {
			c.suggestions = append(c.suggestions, cmd)
			continue
		}
		// Check aliases
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(strings.ToLower(alias), query) {
				c.suggestions = append(c.suggestions, cmd)
				break
			}
		}
	}
}

// parseCommand parses the current input into a command result
func (c *CommandPalette) parseCommand() *CommandResult {
	value := strings.TrimSpace(c.input.Value())
	if value == "" {
		return nil
	}

	parts := strings.Fields(value)
	cmdName := strings.ToLower(parts[0])

	// Resolve aliases
	resolvedCmd := cmdName
	for _, cmd := range AvailableCommands {
		if cmd.Name == cmdName {
			resolvedCmd = cmd.Name
			break
		}
		for _, alias := range cmd.Aliases {
			if alias == cmdName {
				resolvedCmd = cmd.Name
				break
			}
		}
	}

	result := &CommandResult{
		Command: resolvedCmd,
	}
	if len(parts) > 1 {
		result.Args = parts[1:]
	}

	return result
}

// View renders the command palette
func (c *CommandPalette) View() string {
	if !c.active {
		return ""
	}

	promptStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderFocus).
		Padding(0, 1).
		Width(min(60, c.width-4))

	suggestionStyle := lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	selectedSuggestionStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim)

	var content strings.Builder

	// Prompt and input
	content.WriteString(promptStyle.Render(":"))
	content.WriteString(c.input.View())
	content.WriteString("\n")

	// Suggestions
	if len(c.suggestions) > 0 {
		content.WriteString("\n")
		maxShow := min(6, len(c.suggestions))
		for i := 0; i < maxShow; i++ {
			cmd := c.suggestions[i]
			if i == 0 {
				content.WriteString(selectedSuggestionStyle.Render(cmd.Name))
			} else {
				content.WriteString(suggestionStyle.Render(cmd.Name))
			}
			content.WriteString(" ")
			content.WriteString(descStyle.Render(cmd.Description))
			if i < maxShow-1 {
				content.WriteString("\n")
			}
		}
		if len(c.suggestions) > maxShow {
			content.WriteString("\n")
			content.WriteString(descStyle.Render("...and more"))
		}
	}

	return boxStyle.Render(content.String())
}

// CommandBindings returns key bindings for command mode
func CommandBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "enter", Desc: "execute"},
		{Key: "tab", Desc: "complete"},
		{Key: "esc", Desc: "cancel"},
	}
}
