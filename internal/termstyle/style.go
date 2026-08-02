package termstyle

import (
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type Style string

const (
	Bold    Style = "1"
	Accent  Style = "1;38;2;210;168;255"
	Info    Style = "1;38;2;121;192;255"
	Success Style = "1;38;2;86;211;100"
	Warning Style = "1;38;2;227;179;65"
	Danger  Style = "1;38;2;255;123;114"
	Muted   Style = "38;2;139;148;158"
)

func Enabled(w io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	if value := os.Getenv("CLICOLOR_FORCE"); value != "" && value != "0" {
		return true
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func Paint(w io.Writer, style Style, text string) string {
	if text == "" || !Enabled(w) {
		return text
	}
	return "\x1b[" + string(style) + "m" + text + "\x1b[0m"
}
