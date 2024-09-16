package terminal

import (
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/service"
	"sort"
	"strings"
)

type callContext struct {
	// Prefix     cmdPrefix
	// Scope      api.EvalScope
	// Breakpoint *api.Breakpoint
}

// type cmdfunc func(t *Term, ctx callContext, args string) error
type cmdfunc func(t *Term, ctx callContext, args map[string]interface{}) error

type command struct {
	aliases []string
	// group           commandGroup
	helpMsg string
	cmdFn   cmdfunc
	flags   []Flag
}

func (c command) exec(term *Term, ctx callContext, argstr string) error {
	argmap := make(map[string]interface{})
	for _, f := range c.flags {
		name, val := f.Parse(argstr)
		argmap[name] = val
	}
	return c.cmdFn(term, ctx, argmap)
}

// Returns true if the command matches one of the aliases for this command
func (c command) match(cmdstr string) bool {
	for _, v := range c.aliases {
		if v == cmdstr {
			return true
		}
	}
	return false
}

type Commands struct {
	cmds   []command
	client service.Client
}

// byFirstAlias will sort by the first
// alias of a command.
type byFirstAlias []command

func (a byFirstAlias) Len() int           { return len(a) }
func (a byFirstAlias) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFirstAlias) Less(i, j int) bool { return a[i].aliases[0] < a[j].aliases[0] }

func DebugCommands(client service.Client) *Commands {
	c := &Commands{client: client}
	c.cmds = []command{
		{
			aliases: []string{".reg", ".r"}, cmdFn: execReg,
			flags: []Flag{
				FlagBase[uint64]{"expand", []string{"--expand", "-e"}, 0},
				FlagBase[vm.OpCode]{"opcode", []string{"--opcode", "-o"}, 0},
			},
		},
	}

	sort.Sort(byFirstAlias(c.cmds))
	return c
}

func execReg(t *Term, ctx callContext, args map[string]interface{}) error {
	if val, ok := args["expand"].(uint64); ok {
		res, err := t.client.RegExpand(val)
		if err != nil {
			return err
		}
		t.stdout.Echo(res)
	}

	if val, ok := args["opcode"].(vm.OpCode); ok {
		res, err := t.client.RegOpcode(val)
		if err != nil {
			return err
		}
		t.stdout.Echo(res)
	}

	return nil
}

func (c *Commands) Find(cmdstr string) command {
	for _, cmd := range c.cmds {
		if cmd.match(cmdstr) {
			return cmd
		}
	}
	return command{}
}

// CallWithContext takes a command and a context that command should be executed in.
func (c *Commands) CallWithContext(cmdstr string, t *Term, ctx callContext) error {
	vals := strings.SplitN(strings.TrimSpace(cmdstr), " ", 2)
	cmdname := vals[0]
	var args string
	if len(vals) > 1 {
		args = vals[1]
	}
	return c.Find(cmdname).exec(t, ctx, args)
}

// Call takes a command to execute.
func (c *Commands) Call(cmdstr string, t *Term) error {
	ctx := callContext{}
	return c.CallWithContext(cmdstr, t, ctx)
}
