package so

import (
	"fmt"
	"os"
)

// LoadBriefing returns the contents of ~/.config/so/briefing.md.
// Callers should call EnsureDefaults() first if they want the file to
// be auto-created.
func LoadBriefing() (string, error) {
	b, err := os.ReadFile(BriefingPath())
	if err != nil {
		return "", fmt.Errorf("read briefing: %w", err)
	}
	return string(b), nil
}
