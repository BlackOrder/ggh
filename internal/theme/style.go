package theme

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var BaseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var HeaderStyle = table.DefaultStyles().Header.
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240")).
	BorderBottom(true).
	Bold(false)

var SelectedStyle = table.DefaultStyles().Selected.
	Foreground(lipgloss.Color("229")).
	Background(lipgloss.Color("57")).
	Bold(false)
