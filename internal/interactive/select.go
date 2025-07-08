package interactive

import (
	"fmt"
	"github.com/byawitz/ggh/internal/config"
	"github.com/byawitz/ggh/internal/history"
	"github.com/byawitz/ggh/internal/settings"
	"github.com/byawitz/ggh/internal/theme"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	table        table.Model
	allRows      []table.Row
	filteredRows []table.Row
	filtering    bool
	filterText   string
	choice       config.SSHConfig
	exit         bool
	windowWidth  int
	windowHeight int
	tableWidth   int
	tableHeight  int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	// 1. Handle window resize events
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		w, h, cols := theme.AdjustTableDimensions(
			m.table.Columns(),
			m.windowWidth,
			m.windowHeight,
		)

		m.tableWidth = w
		m.tableHeight = h

		// Apply the new widths
		m.table.SetColumns(cols)
		m.resetTableHeight()

		if settings.Get().Fullscreen {
			return m, tea.EnterAltScreen
		}

		return m, tea.ExitAltScreen

	case tea.KeyMsg:
		if m.filtering {
			switch msg.Type {
			case tea.KeyRunes:
				// Add the typed character to the filter text
				m.filterText += string(msg.Runes)
				m.applyFilter()
				return m, nil
			case tea.KeyBackspace:
				// Remove the last character from the filter text
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.applyFilter()
				}
				return m, nil
			default:
				// any other keys, pass to the table
			}
		}
		switch msg.String() {
		case "/":
			if !m.filtering {
				m.filtering = true
				m.filterText = ""
				m.applyFilter()
				return m, nil
			}
			// If we are already filtering, we don't want to do anything
			return m, nil
		case "d":
			selectedRow := m.table.SelectedRow()
			// guard against selection nil
			if selectedRow == nil {
				return m, nil
			}
			history.RemoveByIP(selectedRow)

			// Filter out the selected row and all rows with the same IP/host
			rows := []table.Row{}
			for _, row := range m.table.Rows() {
				if row[1] != selectedRow[1] {
					rows = append(rows, row)
				}
			}
			m.allRows = rows
			m.table.SetRows(m.allRows)

			m.table, cmd = m.table.Update("") // Overrides default `d` behavior

			// check if the table is empty
			if len(m.table.Rows()) == 0 {
				m.exit = true
				return m, tea.Quit
			}

			return m, cmd
		case "r":
			selectedRow := m.table.SelectedRow()
			// guard against selection nil
			if selectedRow == nil {
				return m, nil
			}
			history.RemoveByName(selectedRow)

			// Filter out the selected row
			rows := []table.Row{}
			for _, row := range m.table.Rows() {
				if row[0] != selectedRow[0] {
					rows = append(rows, row)
				}
			}
			m.allRows = rows
			m.table.SetRows(m.allRows)

			m.table, cmd = m.table.Update("")

			// check if the table is empty
			if len(m.table.Rows()) == 0 {
				m.exit = true
				return m, tea.Quit
			}

			return m, cmd
		case "w":
			// toggle fullscreen mode
			newsettings := settings.Get()
			newsettings.Fullscreen = !newsettings.Fullscreen
			if err := settings.Save(newsettings); err == nil {
				m.resetTableHeight()

				if newsettings.Fullscreen {
					return m, tea.EnterAltScreen
				}

				return m, tea.ExitAltScreen
			}

			// If we can't save the settings, do nothing
			return m, nil
		case "q", "ctrl+c", "esc":
			if m.filtering {
				m.stopFiltering()
				return m, nil
			}
			m.exit = true
			return m, tea.Quit
		case "enter":
			selectedRow := m.table.SelectedRow()
			// guard against selection nil
			if selectedRow == nil {
				return m, nil
			}
			m.choice = setConfig(selectedRow)
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// applyFilter re-filters the "allRows" into "filteredRows" based on m.filterText
func (m *model) applyFilter() {
	if m.filterText == "" {
		// no filter → show all
		m.filteredRows = m.allRows
	} else {
		var out []table.Row
		lowerFilter := strings.ToLower(m.filterText)
		for _, row := range m.allRows {
			// For example, we check row[0] or row[1], or all fields
			rowStr := strings.ToLower(strings.Join(row, " "))
			if strings.Contains(rowStr, lowerFilter) {
				out = append(out, row)
			}
		}
		m.filteredRows = out
	}
	m.table.SetRows(m.filteredRows)
}

// stopFiltering leaves filtering mode & restores all data
func (m *model) stopFiltering() {
	m.filtering = false
	m.filterText = ""
	m.filteredRows = m.allRows
	m.table.SetRows(m.filteredRows)
}

func setConfig(row table.Row) config.SSHConfig {
	return config.SSHConfig{
		Name: row[0],
		Host: row[1],
		Port: row[2],
		User: row[3],
		Key:  row[4],
	}
}

func (m model) View() string {
	if m.choice.Host != "" || m.exit {
		return ""
	}

	if m.tableHeight < 3 {
		msg := fmt.Sprintf(
			"Too small (%d lines). Need ≥ 6 lines.",
			m.windowHeight,
		)

		// Style: high-contrast foreground, rounded border, padded.
		styled := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#FF0060", Dark: "#FF79C6"}).
			Render(msg)

		// Center horizontally by calculating left padding.
		pad := max((m.windowWidth-lipgloss.Width(styled))/2, 0)
		centered := strings.Repeat(" ", pad) + styled

		return centered
	}

	return theme.BaseStyle.Render(m.table.View()) + "\n" + m.HelpView() // ← exactly one row
}

func (m model) HelpView() string {
	var blocks []string
	km := table.DefaultKeyMap()
	if m.filtering {
		blocks = append(blocks, "esc quit • ")
	} else {

		blocks = append(blocks,
			fmt.Sprintf("%s %s", km.LineUp.Help().Key, km.LineUp.Help().Desc),
			fmt.Sprintf("%s %s", km.LineDown.Help().Key, km.LineDown.Help().Desc),
		)

		if len(m.table.Columns()) == 6 {
			blocks = append(blocks, "d delete", "r remove")
		}

		blocks = append(blocks,
			"w window/full",
			"/ filter",
			"q quit",
		)
	}

	// join with “ • ” and colour once
	help := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#B2B2B2", Dark: "#4A4A4A"}).
		Render(strings.Join(blocks, " • "))

	if m.filtering {
		prompt := lipgloss.NewStyle().
			Foreground(lipgloss.Color("57")).
			Render("/" + m.filterText)
		return " " + help + prompt
	}
	return " " + help
}

func (m *model) resetTableHeight() {
	m.table.SetHeight(theme.GetTableHeight(
		m.tableHeight,
		len(m.filteredRows),
	))
	m.table.SetWidth(m.tableWidth)
}

func Select(rows []table.Row, what theme.TableStyle) config.SSHConfig {
	t := table.New(
		table.WithColumns(theme.GetColumns(what)),
		table.WithRows(rows),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = theme.HeaderStyle
	s.Selected = theme.SelectedStyle

	t.SetStyles(s)
	_m := model{
		table:        t,
		allRows:      rows,
		filteredRows: rows,
	}

	var p *tea.Program
	if settings.Get().Fullscreen {
		p = tea.NewProgram(_m, tea.WithAltScreen())
	} else {
		p = tea.NewProgram(_m)
	}
	m, err := p.Run()
	if err != nil {
		fmt.Println("error while running the interactive selector, ", err)
		os.Exit(1)
	}
	// Assert the final tea.Model to our local model and print the choice.
	if m, ok := m.(model); ok {
		if m.choice.Host != "" {
			return m.choice
		}
		if m.exit {
			os.Exit(0)
		}
	}

	return config.SSHConfig{}
}
