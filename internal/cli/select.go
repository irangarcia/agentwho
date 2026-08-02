package cli

import (
	"bufio"
	"errors"

	"github.com/irangarcia/agentwho/internal/tui"
)

type menuOption = tui.Option

func (a *app) selectOne(reader *bufio.Reader, title string, options []menuOption, defaultIndex int) (string, error) {
	value, err := tui.SelectOne(reader, a.stdinFile, a.out, title, options, defaultIndex)
	if errors.Is(err, tui.ErrCancelled) {
		return "", silent(err)
	}
	return value, err
}
