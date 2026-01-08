package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	// Navigation
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Back   key.Binding
	Top    key.Binding
	Bottom key.Binding

	// Actions
	Refresh        key.Binding
	Filter         key.Binding
	Logs           key.Binding
	CloudWatchLogs key.Binding
	Help           key.Binding
	Quit           key.Binding
	PortForward    key.Binding
	Tunnels        key.Binding
	StopTunnel     key.Binding
	RestartTunnel  key.Binding
	ClearTunnels   key.Binding
	LambdaInvoke   key.Binding

	// Log scrolling
	LogScrollUp   key.Binding
	LogScrollDown key.Binding
	LogScrollEnd  key.Binding

	// Filter mode
	FilterAccept key.Binding
	FilterClear  key.Binding

	// Copy mode
	CopyMode      key.Binding
	YankClipboard key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "back"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "select"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		CloudWatchLogs: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "cloudwatch logs"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		PortForward: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "port forward"),
		),
		Tunnels: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tunnels"),
		),
		StopTunnel: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop tunnel"),
		),
		RestartTunnel: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart tunnel"),
		),
		ClearTunnels: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear terminated"),
		),
		LambdaInvoke: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "invoke"),
		),
		LogScrollUp: key.NewBinding(
			key.WithKeys("K", "pgup"),
			key.WithHelp("K/PgUp", "scroll logs up"),
		),
		LogScrollDown: key.NewBinding(
			key.WithKeys("J", "pgdown"),
			key.WithHelp("J/PgDn", "scroll logs down"),
		),
		LogScrollEnd: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "scroll logs to end"),
		),
		FilterAccept: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "accept"),
		),
		FilterClear: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear"),
		),
		CopyMode: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy mode"),
		),
		YankClipboard: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank to clipboard"),
		),
	}
}

// ShortHelp returns keybindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Filter, k.Logs, k.Quit}
}

// FullHelp returns keybindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.Enter, k.Back, k.Refresh},
		{k.Filter, k.Logs, k.Help, k.Quit},
	}
}
