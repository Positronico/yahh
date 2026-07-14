// Package shellhook renders the shell integration scripts printed by
// `yahh init`. The templates are static; the only interpolated value is
// the shell-quoted path of the yahh binary.
package shellhook

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

// Params configures the rendered init script.
type Params struct {
	BinPath     string // absolute path to the yahh binary, NOT yet quoted
	AutoClean   bool   // trigger a throttled background clean at startup
	Completions bool   // source `yahh completion <shell>` when available
}

// Render produces the init script for "zsh" or "bash".
func Render(shell string, p Params) (string, error) {
	if shell != "zsh" && shell != "bash" {
		return "", fmt.Errorf("unsupported shell %q (supported: zsh, bash)", shell)
	}
	tmpl, err := template.ParseFS(templates, "templates/yahh."+shell+".tmpl")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = tmpl.Execute(&b, struct {
		BinPath     string
		AutoClean   bool
		Completions bool
	}{Quote(p.BinPath), p.AutoClean, p.Completions})
	return b.String(), err
}

// Quote single-quotes s for safe interpolation into shell code.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
