package main

import (
	"fmt"
	"os"

	"github.com/irangarcia/agentwho/internal/cli"
	"github.com/irangarcia/agentwho/internal/termstyle"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		if !cli.IsSilent(err) {
			fmt.Fprintln(os.Stderr, termstyle.Paint(os.Stderr, termstyle.Danger, "Error:"), err)
		}
		os.Exit(1)
	}
}
