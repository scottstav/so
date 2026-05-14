package so

import (
	"os"
	"testing"
)

// TestMain isolates this package's tests from the developer's environment.
// SO_AGENTS_CONF in particular leaks into AgentsConfPath() and would cause
// tests using XDG_CONFIG_HOME to write to (and clobber) the real override
// file instead of the temp dir. Tests that deliberately exercise the
// override still set it via t.Setenv.
func TestMain(m *testing.M) {
	os.Unsetenv("SO_AGENTS_CONF")
	os.Exit(m.Run())
}
