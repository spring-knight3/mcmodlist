package cli

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

type execClipboard struct{}

func (execClipboard) Copy(text string) error {
	cmd, err := clipboardCommand()
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func clipboardCommand() (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("clip"), nil
	case "darwin":
		return exec.Command("pbcopy"), nil
	default:
		for _, tool := range []string{"wl-copy", "xclip", "xsel"} {
			if _, err := exec.LookPath(tool); err != nil {
				continue
			}
			switch tool {
			case "xclip":
				return exec.Command(tool, "-selection", "clipboard"), nil
			case "xsel":
				return exec.Command(tool, "--clipboard", "--input"), nil
			default:
				return exec.Command(tool), nil
			}
		}
	}
	return nil, errors.New("no clipboard tool found (install wl-clipboard, xclip, or xsel)")
}
