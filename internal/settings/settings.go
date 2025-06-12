package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Fullscreen bool `json:"fullscreen"`
}

func FetchWithDefaultFile() Settings {
	return Fetch(getFile())
}

func Fetch(file []byte) Settings {
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

func Save(s Settings) (*Settings, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return &s, os.WriteFile(getFileLocation(), b, 0644)
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
