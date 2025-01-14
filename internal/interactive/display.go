package interactive

import (
	"fmt"
	"github.com/byawitz/ggh/internal/config"
	"github.com/byawitz/ggh/internal/history"
	"github.com/byawitz/ggh/internal/ssh"
	"github.com/charmbracelet/bubbles/table"
	"log"
	"os"
	"time"
)

func Config(value string) []string {
	list, err := config.ParseWithSearch(value, config.GetConfigFile())
	if err != nil || len(list) == 0 {
		fmt.Println("No config found.")
		os.Exit(0)
	}

	var rows []table.Row
	for _, c := range list {
		rows = append(rows, table.Row{
			c.Name,
			c.Host,
			c.Port,
			c.User,
			c.Key,
		})
	}
	c := Select(rows, SelectConfig)
	return ssh.GenerateCommandArgs(c)
}

func History() []string {
	list, err := history.FetchWithDefaultFile()

	if err != nil {
		log.Fatal(err)
	}

	if len(list) == 0 {
		fmt.Println("No history found.")
		os.Exit(0)
	}

	// clean list for duplicates, keep the latest
	uniqueMap := make(map[string]history.SSHHistory) // Store the full struct instead of just the date
	for _, history := range list {
		if existing, ok := uniqueMap[history.Connection.Host]; !ok || history.Date.After(existing.Date) {
			uniqueMap[history.Connection.Host] = history // Update with the latest history entry
		}
	}

	// Extract the unique values from the map
	uniqueList := make([]history.SSHHistory, 0, len(uniqueMap))
	for _, history := range uniqueMap {
		uniqueList = append(uniqueList, history)
	}

	var rows []table.Row
	currentTime := time.Now()
	for _, historyItem := range uniqueList {
		rows = append(rows, table.Row{
			historyItem.Connection.Name,
			historyItem.Connection.Host,
			historyItem.Connection.Port,
			historyItem.Connection.User,
			historyItem.Connection.Key,
			history.ReadableTime(currentTime.Sub(historyItem.Date)),
		})
	}
	c := Select(rows, SelectHistory)
	return ssh.GenerateCommandArgs(c)
}
