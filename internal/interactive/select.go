package interactive

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/blackorder/ggh/internal/config"
	"github.com/blackorder/ggh/internal/history"
	"github.com/blackorder/ggh/internal/theme"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Selecting int

const (
	SelectConfig Selecting = iota
	SelectHistory
)

// The model now tracks window width/height, filtering mode, etc.
type model struct {
	table        table.Model
	allRows      []table.Row
	filteredRows []table.Row
	filtering    bool
	filterText   string
	choice       config.SSHConfig
	what         Selecting
	exit         bool
	windowWidth  int
	windowHeight int
	mOutput      string // exit message
}

func (m model) Init() tea.Cmd {
	// Clear the screen at startup
	return tea.ClearScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	// 1. Handle window resize events
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		// Make some margin for help text at the bottom or other elements
		widthForTable := m.windowWidth - 3
		if widthForTable < 3 {
			widthForTable = 3
		}
		// Extra margin for content
		widthForTableContent := widthForTable - 13
		heightForTable := m.windowHeight - 5
		if heightForTable < 5 {
			heightForTable = 5
		}

		cols := m.table.Columns()

		switch m.what {
		// SELECT CONFIG
		case SelectConfig:
			// columns = [Name, Host, Port, User, Key]
			// base widths = 15,20,5,10,10 = total 60
			baseWidths := []int{15, 20, 5, 10, 10}
			const totalBase = 60

			if widthForTableContent >= totalBase {
				leftover := widthForTableContent - totalBase
				leftoverForKey := 0
				leftoverForName := 0

				for leftover > 0 {
					if leftoverForKey < 15 {
						leftoverForKey++
						leftover--
					} else if leftoverForKey < 30 && leftover > 1 {
						leftoverForName++
						leftoverForKey++
						leftover -= 2
					} else {
						leftoverForName++
						leftover--
					}
				}

				cols[0].Width = baseWidths[0] + leftoverForName // Name
				cols[1].Width = baseWidths[1]                   // Host
				cols[2].Width = baseWidths[2]                   // Port
				cols[3].Width = baseWidths[3]                   // User
				cols[4].Width = baseWidths[4] + leftoverForKey  // Key
			} else {
				// Scale all columns proportionally
				ratio := float64(widthForTableContent) / float64(totalBase)
				for i := range cols {
					w := int(math.Round(float64(baseWidths[i]) * ratio))
					if w < 1 {
						w = 1
					}
					cols[i].Width = w
				}
			}

		// SELECT HISTORY
		case SelectHistory:
			// columns = [Name,Host,Port,User,Key,Last login]
			// base widths = 10,20,5,10,0,15 = total 60
			baseWidths := []int{10, 20, 5, 10, 0, 15}
			const totalBase = 60

			if widthForTableContent >= totalBase {
				leftover := widthForTableContent - totalBase
				leftoverForKey := 0
				leftoverForName := 0

				for leftover > 0 {
					if leftoverForKey < 15 {
						leftoverForKey++
						leftover--
					} else if leftoverForKey < 30 && leftover > 1 {
						leftoverForName++
						leftoverForKey++
						leftover -= 2
					} else {
						leftoverForName++
						leftover--
					}
				}

				cols[0].Width = baseWidths[0] + leftoverForName // Name
				cols[1].Width = baseWidths[1]                   // Host
				cols[2].Width = baseWidths[2]                   // Port
				cols[3].Width = baseWidths[3]                   // User
				cols[4].Width = baseWidths[4] + leftoverForKey  // Key
				cols[5].Width = baseWidths[5]                   // Last login
			} else {
				// Not enough space → scale all columns proportionally
				ratio := float64(widthForTableContent) / float64(totalBase)
				for i := range cols {
					w := int(math.Round(float64(baseWidths[i]) * ratio))
					if w < 1 {
						w = 1
					}
					cols[i].Width = w
				}
			}
		}

		// Apply the new widths
		m.table.SetColumns(cols)
		m.table.SetWidth(widthForTable)
		m.table.SetHeight(heightForTable)

		return m, nil

	// 3. Handle key events
	case tea.KeyMsg:
		if m.filtering {
			switch msg.Type {
			case tea.KeyRunes:
				// Add typed character(s) to filterText
				m.filterText += string(msg.Runes)
				m.applyFilter()
				return m, nil

			case tea.KeyBackspace:
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.applyFilter()
				}
				return m, nil

			// Pressing Enter in filtering mode → do nothing special
			// (We *could* also exit filtering mode if you wanted.)
			case tea.KeyEnter:
				// We still let the table handle selection.
				// so let's break out and do the normal table update below
				break

			// Pressing 'esc' in filtering mode → exit filtering & restore data
			case tea.KeyEsc:
				fallthrough
			case tea.KeyCtrlC:
				fallthrough
			default:
				// any other keys, pass to the table
			}
		}

		switch msg.String() {

		case "/":
			if !m.filtering {
				// Enter filtering mode
				m.filtering = true
				m.filterText = "" // or keep existing text if you want incremental
				m.applyFilter()   // apply empty filter or partial filter
				return m, nil
			}
			// If we’re *already* filtering, ignore
			return m, nil
		case "d":
			if m.table.SelectedRow() == nil {
				return m, nil
			}
			history.RemoveByIP(m.table.SelectedRow())

			// Get the currently selected row
			selectedRow := m.table.SelectedRow()
			hostToRemove := selectedRow[1]

			var newAllRows []table.Row
			for _, row := range m.allRows {
				// If this row’s host does not match the one to remove, we keep it
				if row[1] != hostToRemove {
					newAllRows = append(newAllRows, row)
				}
			}
			m.allRows = newAllRows

			// Update the table with the new rows
			m.table.SetRows(m.allRows)

			m.table, cmd = m.table.Update("") // Overrides default `d` behavior

			// if table is empty, exit
			if len(m.table.Rows()) == 0 {
				m.exit = true
				m.mOutput = "No history found."
				return m, tea.Quit
			}

			return m, cmd
		case "r":
			if m.table.SelectedRow() == nil {
				return m, nil
			}
			history.RemoveByName(m.table.SelectedRow())

			// Get the currently selected row
			selectedRow := m.table.SelectedRow()
			nameToRemove := selectedRow[0]

			var newallRows []table.Row
			for _, row := range m.allRows {
				// If this row’s host does not match the one to remove, we keep it
				if row[0] != nameToRemove {
					newallRows = append(newallRows, row)
				}
			}
			m.allRows = newallRows

			// Update the table with the new rows
			m.table.SetRows(m.allRows)

			m.table, cmd = m.table.Update("") // Overrides default `d` behavior

			// if table is empty, exit
			if len(m.table.Rows()) == 0 {
				m.exit = true
				m.mOutput = "No history found."
				return m, tea.Quit
			}

			return m, cmd
		case "q", "ctrl+c", "esc":
			if m.filtering {
				// If we’re filtering, pressing esc/q just stops filtering
				m.stopFiltering()
				return m, nil
			}
			// else if not filtering, exit the program
			m.exit = true
			return m, tea.Quit
		case "enter":
			if m.table.SelectedRow() == nil {
				return m, nil
			}
			m.choice = setConfig(m.table.SelectedRow())
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

	// If we are filtering, show a small prompt with the filter text
	if m.filtering {
		prompt := lipgloss.NewStyle().
			Background(lipgloss.Color("#87CEEB")).
			Foreground(lipgloss.Color("#000000")).
			Width(m.windowWidth - 5).
			Render(fmt.Sprintf("/%s", m.filterText))
		return theme.BaseStyle.Render(m.table.View()) + "\n  " + prompt + "\n  " + m.HelpFilterView() + "\n"
	}

	return theme.BaseStyle.Render(m.table.View()) + "\n\n  " + m.HelpView() + "\n"
}

// Updated Select function: store `rows` in both allRows & filteredRows
func Select(rows []table.Row, what Selecting) config.SSHConfig {
	var columns []table.Column

	if what == SelectConfig {
		columns = []table.Column{
			{Title: "Name"}, // index 0
			{Title: "Host"}, // index 1
			{Title: "Port"}, // index 2
			{Title: "User"}, // index 3
			{Title: "Key"},  // index 4
		}
	} else if what == SelectHistory {
		columns = []table.Column{
			{Title: "Name"},       // index 0
			{Title: "Host"},       // index 1
			{Title: "Port"},       // index 2
			{Title: "User"},       // index 3
			{Title: "Key"},        // index 4
			{Title: "Last login"}, // index 5
		}
	} else {
		fmt.Println("Invalid selection type")
		os.Exit(1)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),

		// 3. We can still set an initial height if the terminal size is unknown.
		//    If we get a WindowSizeMsg, we’ll update this dynamically in the model's Update().
		table.WithHeight(int(math.Min(8, float64(len(rows)+1)))),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)

	t.SetStyles(s)

	// Create and run the Bubble Tea program,
	// storing rows in allRows + filteredRows.
	initialModel := model{
		table:        t,
		allRows:      rows,
		filteredRows: rows, // Start unfiltered
		what:         what,
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())
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
		if m.mOutput != "" {
			fmt.Println(m.mOutput)
		}
		if m.exit {
			os.Exit(0)
		}
	}

	return config.SSHConfig{}
}

func (m model) HelpView() string {

	km := table.DefaultKeyMap()

	var b strings.Builder

	b.WriteString(generateHelpBlock(km.LineUp.Help().Key, km.LineUp.Help().Desc, true))
	b.WriteString(generateHelpBlock(km.LineDown.Help().Key, km.LineDown.Help().Desc, true))

	if m.what == SelectHistory {
		b.WriteString(generateHelpBlock("d", "delete (Host)", true))
		b.WriteString(generateHelpBlock("r", "delete (Name)", true))
	}
	b.WriteString(generateHelpBlock("/", "filter mode", true))
	b.WriteString(generateHelpBlock("q/esc", "quit", false))

	return b.String()
}

func (m model) HelpFilterView() string {

	km := table.DefaultKeyMap()

	var b strings.Builder

	b.WriteString(generateHelpBlock(km.LineUp.Help().Key, km.LineUp.Help().Desc, true))
	b.WriteString(generateHelpBlock(km.LineDown.Help().Key, km.LineDown.Help().Desc, true))
	b.WriteString(generateHelpBlock("esc", "exit filter mode", false))

	return b.String()
}

func generateHelpBlock(key, desc string, withSep bool) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#909090",
		Dark:  "#626262",
	})

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#4A4A4A",
	})

	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#DDDADA",
		Dark:  "#3C3C3C",
	})

	sep := sepStyle.Inline(true).Render(" • ")

	str := keyStyle.Inline(true).Render(key) +
		" " +
		descStyle.Inline(true).Render(desc)

	if withSep {
		str += sep
	}

	return str
}
