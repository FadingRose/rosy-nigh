package terminal

import (
	"bufio"
	"fadingrose/rosy-nigh/service"
	"fmt"
	"io"
	"os"
	"strings"
)

// Term represents the daemon terminal, using RPC to communicate with the FuzzHost
type Term struct {
	client service.Client
	cmds   *Commands
	stdout *transcriptWriter
	prompt string

	input *bufio.Reader
}

func NewTerminal(client service.Client, stdin io.Reader) *Term {
	cmds := DebugCommands(client)
	t := &Term{
		client: client,
		cmds:   cmds,
		stdout: &transcriptWriter{w: os.Stdout},
		prompt: "(rosy-nigh) ",
		input:  bufio.NewReader(stdin),
	}
	return t
}

func (t *Term) Run() error {
	var lastCmd string
	for {
		cmdstr, err := t.promptFromInput()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("exiting terminal: %v", err)
			}
			return fmt.Errorf("error reading input: %v", err)
		}

		t.stdout.Echo(t.prompt + cmdstr + "\n")

		if strings.TrimSpace(cmdstr) == "" {
			cmdstr = lastCmd
		}

		lastCmd = cmdstr

		if err := t.cmds.Call(cmdstr, t); err != nil {
			fmt.Println(err)
			// return fmt.Errorf("error executing command: %v", err)
		}
	}
	return nil
}

// promptFromInput reads a line of input from the terminal
// TODO: support auto-completion
func (t *Term) promptFromInput() (string, error) {
	return t.input.ReadString('\n')
}
