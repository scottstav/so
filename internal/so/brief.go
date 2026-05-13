package so

import (
	"fmt"
	"io"
)

// Brief writes the contents of briefing.md to w. Used by the `so brief`
// subcommand so an agent can pull the briefing on demand (e.g. when
// resumed and the briefing wasn't injected on launch).
func Brief(w io.Writer) error {
	if err := EnsureDefaults(); err != nil {
		return err
	}
	text, err := LoadBriefing()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, text)
	return err
}
