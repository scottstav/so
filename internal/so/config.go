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

// AgentsConfPath returns the path used to READ the agent registry.
// Honors SO_AGENTS_CONF for swapping in alternate registries (e.g. a
// personal one); falls back to the canonical default.
func AgentsConfPath() string {
	if p := os.Getenv("SO_AGENTS_CONF"); p != "" {
		return p
	}
	return defaultAgentsConfPath()
}

// defaultAgentsConfPath is the canonical bootstrap location, never
// affected by env overrides. EnsureDefaults always writes here.
func defaultAgentsConfPath() string { return filepath.Join(ConfigDir(), "agents.conf") }

// BriefingPath returns the path to briefing.md.
func BriefingPath() string { return filepath.Join(ConfigDir(), "briefing.md") }

// EnsureDefaults writes the default agents.conf and briefing.md to the
// canonical paths if they don't already exist. Never overwrites, and
// never writes to an SO_AGENTS_CONF override location — if you point
// the override at a custom path, you maintain that file yourself.
func EnsureDefaults() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := copyDefaultIfMissing("defaults/agents.conf", defaultAgentsConfPath()); err != nil {
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
