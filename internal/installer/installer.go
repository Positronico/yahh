// Package installer manages the yahh init block in shell rc files.
package installer

import (
	"fmt"
	"os"
	"strings"
)

const (
	// BeginMarker and EndMarker delimit the managed block in rc files.
	BeginMarker = "# >>> yahh init >>>"
	EndMarker   = "# <<< yahh init <<<"
)

// Block returns the managed rc-file block for a shell. The command -v
// guard keeps the rc file working if yahh is uninstalled from PATH.
func Block(shell string) string {
	return fmt.Sprintf("%s\ncommand -v yahh >/dev/null 2>&1 && eval \"$(yahh init %s)\"\n%s\n",
		BeginMarker, shell, EndMarker)
}

// Install appends the managed block to rcPath (creating the file if
// missing). Returns false if the block is already present.
func Install(rcPath, shell string) (bool, error) {
	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if strings.Contains(string(content), BeginMarker) {
		return false, nil
	}
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	prefix := "\n"
	if len(content) == 0 || strings.HasSuffix(string(content), "\n\n") {
		prefix = ""
	}
	_, err = f.WriteString(prefix + Block(shell))
	return err == nil, err
}

// Uninstall removes the managed block from rcPath. Returns false if no
// block was found (a missing file is not an error).
func Uninstall(rcPath string) (bool, error) {
	content, err := os.ReadFile(rcPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(content), "\n")
	var kept []string
	inBlock, removed := false, false
	for _, line := range lines {
		switch {
		case strings.TrimSpace(line) == BeginMarker:
			inBlock, removed = true, true
		case inBlock && strings.TrimSpace(line) == EndMarker:
			inBlock = false
		case !inBlock:
			kept = append(kept, line)
		}
	}
	if !removed {
		return false, nil
	}
	out := strings.Join(kept, "\n")
	// Collapse the blank line the install step added before the block.
	out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	info, err := os.Stat(rcPath)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = info.Mode()
	}
	return true, os.WriteFile(rcPath, []byte(out), mode)
}
