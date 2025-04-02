package history

import (
	"encoding/json"
	"fmt"
	"github.com/byawitz/ggh/internal/config"
	"github.com/byawitz/ggh/internal/theme"
	"github.com/charmbracelet/bubbles/table"
	"log"
	"time"
)

type SSHHistory struct {
	Connection config.SSHConfig `json:"connection"`
	Date       time.Time        `json:"date"`
}

func FetchWithDefaultFile() ([]SSHHistory, error) {
	return Fetch(getFile())
}

func Fetch(file []byte) ([]SSHHistory, error) {
	var historyList []SSHHistory

	if len(file) == 0 {
		return historyList, nil
	}

	err := json.Unmarshal(file, &historyList)
	if err != nil {
		return nil, err
	}

	search, err := config.ParseWithSearch("", config.GetConfigFile())

	if err != nil {
		return historyList, nil
	}

FormatHistoryLoop:
	for i, historyItem := range historyList {
		if historyItem.Connection.Name == "" {
			historyList[i].Connection.Name = config.DirectSSH
		} else {
			// Check if the name is in the config file
			for _, sshConfig := range search {
				if sshConfig.Name == historyItem.Connection.Name {
					// Update the config
					historyList[i].Connection = sshConfig
					continue FormatHistoryLoop
				}
			}
			// If not, add a missing config prefix
			historyList[i].Connection.Name = fmt.Sprintf("%s%s", config.MissingConfig, historyItem.Connection.Name)
		}
	}

	return historyList, nil
}

func Print() {
	list, err := FetchWithDefaultFile()

	if err != nil {
		log.Fatal(err)
	}

	if len(list) == 0 {
		fmt.Println("No history found.")
		return
	}
	var rows []table.Row
	currentTime := time.Now()
	for _, history := range list {
		rows = append(rows, table.Row{history.Connection.Name,
			history.Connection.Host,
			history.Connection.Port,
			history.Connection.User,
			history.Connection.Key,
			fmt.Sprintf("%s", ReadableTime(currentTime.Sub(history.Date))),
		})
	}

	fmt.Println(theme.PrintTable(rows, theme.PrintHistory))
}

func ReadableTime(d time.Duration) string {
	if d.Seconds() < 60 {
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	}
	if d.Minutes() < 60 {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}

	if d.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}

	if days := int(d.Hours() / 24); days < 90 {
		return fmt.Sprintf("%d days ago", days)
	}

	return "Long time ago"
}
