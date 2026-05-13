package so

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed defaults/agents.conf defaults/briefing.md
var defaultsFS embed.FS

// ConfigDir returns the directory holding so's config files. Honors
// XDG_CONFIG_HOME, falls back to $HOME/.config/so.
func ConfigDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "so")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "so")
}

// AgentsConfPath returns the path to agents.conf.
func AgentsConfPath() string { return filepath.Join(ConfigDir(), "agents.conf") }

// BriefingPath returns the path to briefing.md.
func BriefingPath() string { return filepath.Join(ConfigDir(), "briefing.md") }

// EnsureDefaults writes the default agents.conf and briefing.md if they
// don't already exist. Existing files are never overwritten.
func EnsureDefaults() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := copyDefaultIfMissing("defaults/agents.conf", AgentsConfPath()); err != nil {
		return err
	}
	if err := copyDefaultIfMissing("defaults/briefing.md", BriefingPath()); err != nil {
		return err
	}
	return nil
}

func copyDefaultIfMissing(embedPath, dstPath string) error {
	if _, err := os.Stat(dstPath); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", dstPath, err)
	}
	b, err := defaultsFS.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", embedPath, err)
	}
	if err := os.WriteFile(dstPath, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	return nil
}
