package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/irangarcia/agentwho/internal/termstyle"
	"golang.org/x/term"
)

type Option struct {
	Label       string
	Description string
	Value       string
}

var ErrCancelled = errors.New("selection cancelled")

// SelectOne presents an arrow-key menu in a terminal and a numbered prompt
// when input or output is redirected.
func SelectOne(reader *bufio.Reader, inputFile *os.File, output io.Writer, title string, options []Option, defaultIndex int) (string, error) {
	if len(options) == 0 {
		return "", errors.New("selection has no options")
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}
	if !canUseArrowMenu(inputFile, output) {
		return selectNumbered(reader, output, title, options, defaultIndex)
	}

	_, _ = fmt.Fprintf(output, "\n%s\n\n", termstyle.Paint(output, termstyle.Accent, title))
	_, _ = fmt.Fprintln(output, termstyle.Paint(output, termstyle.Muted, "Use ↑/↓ to move and Enter to select."))

	fd := int(inputFile.Fd())
	previous, err := term.MakeRaw(fd)
	if err != nil {
		return selectNumbered(reader, output, title, options, defaultIndex)
	}
	restored := false
	restore := func() {
		if !restored {
			_ = term.Restore(fd, previous)
			restored = true
		}
	}
	defer restore()

	_, _ = fmt.Fprint(output, "\x1b[?25l")
	defer func() { _, _ = fmt.Fprint(output, "\x1b[?25h") }()

	selected := defaultIndex
	menuLines := renderMenu(output, options, selected, 0)
	for {
		key, err := readMenuKey(inputFile)
		if err != nil {
			restore()
			_, _ = fmt.Fprintln(output)
			return "", fmt.Errorf("read selection: %w", err)
		}
		switch key {
		case menuUp:
			selected = moveSelection(selected, len(options), -1)
			renderMenu(output, options, selected, menuLines)
		case menuDown:
			selected = moveSelection(selected, len(options), 1)
			renderMenu(output, options, selected, menuLines)
		case menuChoose:
			restore()
			_, _ = fmt.Fprintln(output)
			_, _ = fmt.Fprintln(output, termstyle.Paint(output, termstyle.Success, "✓ Selected: "+options[selected].Label))
			return options[selected].Value, nil
		case menuCancel:
			restore()
			_, _ = fmt.Fprintln(output, termstyle.Paint(output, termstyle.Warning, "\nCancelled."))
			return "", ErrCancelled
		}
	}
}

func canUseArrowMenu(input *os.File, output io.Writer) bool {
	if input == nil || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	out, ok := output.(*os.File)
	return ok && term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(out.Fd()))
}

func selectNumbered(reader *bufio.Reader, output io.Writer, title string, options []Option, defaultIndex int) (string, error) {
	if reader == nil {
		return "", errors.New("numbered selection requires an input reader")
	}
	_, _ = fmt.Fprintf(output, "\n%s\n\n", termstyle.Paint(output, termstyle.Accent, title))
	for i, option := range options {
		_, _ = fmt.Fprintf(output, "  %d. %s\n", i+1, option.Label)
		if option.Description != "" {
			fmt.Fprintf(output, "     %s\n", termstyle.Paint(output, termstyle.Muted, option.Description))
		}
		if i < len(options)-1 {
			fmt.Fprintln(output)
		}
	}
	fmt.Fprintf(output, "\nChoice [%d]: ", defaultIndex+1)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = fmt.Sprint(defaultIndex + 1)
	}
	for i, option := range options {
		if answer == fmt.Sprint(i+1) || strings.EqualFold(answer, option.Value) || strings.EqualFold(answer, option.Label) {
			fmt.Fprintln(output, "\n"+termstyle.Paint(output, termstyle.Success, "✓ Selected: "+option.Label))
			return option.Value, nil
		}
	}
	return "", fmt.Errorf("invalid choice %q; choose a number from 1 to %d", answer, len(options))
}

type menuKey int

const (
	menuIgnore menuKey = iota
	menuUp
	menuDown
	menuChoose
	menuCancel
)

func readMenuKey(r io.Reader) (menuKey, error) {
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		return menuIgnore, err
	}
	switch first[0] {
	case '\r', '\n':
		return menuChoose, nil
	case 3, 4:
		return menuCancel, nil
	case 'k', 'K':
		return menuUp, nil
	case 'j', 'J':
		return menuDown, nil
	case 27:
		var sequence [2]byte
		if _, err := io.ReadFull(r, sequence[:]); err != nil {
			return menuIgnore, err
		}
		if sequence[0] == '[' {
			switch sequence[1] {
			case 'A':
				return menuUp, nil
			case 'B':
				return menuDown, nil
			}
		}
	}
	return menuIgnore, nil
}

func moveSelection(current, count, delta int) int {
	if count <= 0 {
		return 0
	}
	return (current + delta + count) % count
}

func renderMenu(w io.Writer, options []Option, selected, previousLines int) int {
	if previousLines > 0 {
		fmt.Fprintf(w, "\x1b[%dA", previousLines)
	}
	lines := 0
	for i, option := range options {
		marker := "  "
		label := option.Label
		if i == selected {
			marker = termstyle.Paint(w, termstyle.Success, "❯ ")
			label = termstyle.Paint(w, termstyle.Success, option.Label)
		}
		fmt.Fprintf(w, "\r\x1b[2K%s%s\r\n", marker, label)
		lines++
		if option.Description != "" {
			fmt.Fprintf(w, "\r\x1b[2K    %s\r\n", termstyle.Paint(w, termstyle.Muted, option.Description))
			lines++
		}
	}
	return lines
}
