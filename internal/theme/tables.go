package theme

import (
	"math"

	"github.com/byawitz/ggh/internal/settings"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

type TableStyle int

const (
	ConfigTable TableStyle = iota
	HistoryTable
)

const (
	marginWidth            = 3
	marginHeight           = 3
	minTableWidth          = 3
	contentExtraMargin     = 12
	preferredKeyExtraWidth = 15
	maxKeyExtraWidth       = 30
	minTableHeight         = 3
	maxTableHeight         = 8
)

func GetColumns(what TableStyle) []table.Column {
	columns := make([]table.Column, 0)

	switch what {
	case ConfigTable:
		columns = append(columns, []table.Column{
			{Title: "Name", Width: 10},
			{Title: "Host", Width: 15},
			{Title: "Port", Width: 10},
			{Title: "User", Width: 10},
			{Title: "Key", Width: 10},
		}...)
	case HistoryTable:
		columns = append(columns, []table.Column{
			{Title: "Name", Width: 10},
			{Title: "Host", Width: 15},
			{Title: "Port", Width: 10},
			{Title: "User", Width: 10},
			{Title: "Key", Width: 10},
			{Title: "Last login", Width: 15},
		}...)
	}

	return columns
}

func PrintTable(rows []table.Row, p TableStyle) string {
	columns := GetColumns(p)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
			Selected: lipgloss.NewStyle(),
		}),
		table.WithHeight(len(rows)+1),
	)

	return BaseStyle.Render(t.View())
}

func GetTableHeight(tableHeight int, filteredRowsLen int) int {
	var expected float64
	max := math.Min(maxTableHeight, float64(tableHeight))
	if settings.Get().Fullscreen {
		// if fullscreen, let the table be as tall as the terminal
		expected = float64(tableHeight)
	} else {
		// if not fullscreen, set the height to a minimum of 1 row, but not more than 8 rows
		expected = float64(filteredRowsLen + 2) // +2 for header and footer
		expected = math.Min(expected, max)
	}

	return int(math.Max(minTableHeight, expected))
}

func AdjustTableDimensions(cols []table.Column, windowWidth int, windowHeight int) (int, int, []table.Column) {
	tableHeight := windowHeight - marginHeight
	tableWidth := max(windowWidth-marginWidth, minTableWidth)
	// Extra margin for content
	widthForTableContent := tableWidth - contentExtraMargin

	switch len(cols) {
	// SELECT CONFIG
	case 5:
		// columns = [Name, Host, Port, User, Key]
		// base widths = 15,20,5,10,10 = total 60
		baseWidths := []int{15, 20, 5, 10, 10}
		const totalBase = 60

		if widthForTableContent >= totalBase {
			leftover := widthForTableContent - totalBase
			leftoverForKey := 0
			leftoverForName := 0

			for leftover > 0 {
				if leftoverForKey < preferredKeyExtraWidth {
					leftoverForKey++
					leftover--
				} else if leftoverForKey < maxKeyExtraWidth && leftover > 1 {
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
				w := max(int(math.Round(float64(baseWidths[i])*ratio)), 1)
				cols[i].Width = w
			}
		}

	// SELECT HISTORY
	case 6:
		// columns = [Name,Host,Port,User,Key,Last login]
		// base widths = 10,20,5,10,0,15 = total 60
		baseWidths := []int{10, 20, 5, 10, 0, 15}
		const totalBase = 60

		if widthForTableContent >= totalBase {
			leftover := widthForTableContent - totalBase
			leftoverForKey := 0
			leftoverForName := 0

			for leftover > 0 {
				if leftoverForKey < preferredKeyExtraWidth {
					leftoverForKey++
					leftover--
				} else if leftoverForKey < maxKeyExtraWidth && leftover > 1 {
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
				w := max(int(math.Round(float64(baseWidths[i])*ratio)), 1)
				cols[i].Width = w
			}
		}
	}

	return tableWidth, tableHeight, cols
}
