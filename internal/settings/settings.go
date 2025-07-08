package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
)

type Settings struct {
	Fullscreen bool `json:"fullscreen"`
}

var S atomic.Value

func init() {
	S.Store(Settings{})

	go func() {
		s := fetchWithDefaultFile()
		S.Store(s)
	}()
}

func Get() Settings {
	if s, ok := S.Load().(Settings); ok {
		return s
	}
	return Settings{}
}

func fetchWithDefaultFile() Settings {
	return fetch(getFile())
}

func fetch(file []byte) Settings {
	var s Settings

	if len(file) == 0 {
		return Settings{}
	}

	err := json.Unmarshal(file, &s)
	if err != nil {
		return Settings{}
	}

	return s
}

func Save(s Settings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(getFileLocation(), b, 0644)

	if err == nil {
		S.Store(s) // Update the atomic value with the new settings
	}

	return err
}

func getFileLocation() string {
	userHomeDir, err := os.UserHomeDir()

	if err != nil {
		return ""
	}

	gghConfigDir := filepath.Join(userHomeDir, ".ggh")

	if err := os.MkdirAll(gghConfigDir, 0700); err != nil {
		return ""
	}

	return filepath.Join(gghConfigDir, "settings.json")

}

func getFile() []byte {

	settings, err := os.ReadFile(getFileLocation())

	if err != nil {
		return []byte{}
	}

	return settings
}
