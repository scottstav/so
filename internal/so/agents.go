package so

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// LoadAgents reads agents.conf from ConfigDir() and returns the registry.
func LoadAgents() (map[string]string, error) {
	f, err := os.Open(AgentsConfPath())
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", AgentsConfPath(), err)
	}
	defer f.Close()
	return parseAgents(f)
}

func parseAgents(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	scan := bufio.NewScanner(r)
	lineno := 0
	for scan.Scan() {
		lineno++
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("line %d: expected key=value, got %q", lineno, line)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if key == "" || val == "" {
			return nil, fmt.Errorf("line %d: empty key or value in %q", lineno, line)
		}
		out[key] = val
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// AgentNames returns the sorted list of registered agent names.
func AgentNames(reg map[string]string) []string {
	names := make([]string, 0, len(reg))
	for k := range reg {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
