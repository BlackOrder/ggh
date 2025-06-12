package settings

import (
	"testing"
	"time"
)

func TestMarshal(t *testing.T) {
	s := FetchWithDefaultFile()
	if s.Fullscreen {
		t.Errorf("expected fullscreen to be false, got %v", s.Fullscreen)
	}

	s.Fullscreen = true
	if _, err := Save(s); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	s = FetchWithDefaultFile()
	if !s.Fullscreen {
		t.Errorf("expected fullscreen to be true, got %v", s.Fullscreen)
	}

	// Reset settings for other tests
	s.Fullscreen = false
	if _, err := Save(s); err != nil {
		t.Fatalf("failed to reset settings: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // Allow time for file operations
}
