package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func RunInteractive(initial Model, in io.Reader, out io.Writer) error {
	model := initial
	scanner := bufio.NewScanner(in)

	for {
		if _, err := fmt.Fprint(out, model.View()); err != nil {
			return fmt.Errorf("write ui view: %w", err)
		}
		if _, err := fmt.Fprint(out, "\ncommand> "); err != nil {
			return fmt.Errorf("write ui prompt: %w", err)
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read ui command: %w", err)
			}
			return nil
		}

		command := strings.ToLower(strings.TrimSpace(scanner.Text()))
		switch command {
		case "q", "quit", "exit":
			return nil
		case "tab", "right", "l":
			model = model.NextTab()
		case "backtab", "left", "h":
			model = model.PrevTab()
		case "1", "2", "3":
			model = model.SelectTab(int(command[0] - '1'))
		case "":
			// No-op; rerender.
		default:
			if _, err := fmt.Fprintf(out, "unknown command: %s\n", command); err != nil {
				return fmt.Errorf("write ui command error: %w", err)
			}
		}
	}
}
