package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
)

var (
	ErrUnimplementedCommand = errors.New("Unimplemented command")
)

type BaseCommand struct {
	FlagSet *flag.FlagSet
	Prefix  string
	name    string
	usage   string
}

func NewBaseCommand(prefix, name, usage string) BaseCommand {
	return BaseCommand{
		FlagSet: flag.NewFlagSet(prefix+" "+name, flag.ContinueOnError),
		Prefix:  prefix,
		name:    name,
		usage:   usage,
	}
}

func (c *BaseCommand) Name() string {
	return c.name
}

func (c *BaseCommand) Usage() string {
	return fmt.Sprintf("Usage of %s %s:\n", c.Prefix, c.name)
	// var b bytes.Buffer
	// output := c.FlagSet.Output()
	// c.FlagSet.SetOutput(&b)
	// c.FlagSet.Usage()
	// c.FlagSet.SetOutput(output)
	// return b.String()
}

func (c *BaseCommand) Synopsis() string {
	return c.usage
}

func (c *BaseCommand) SetFlags(fs *flag.FlagSet) {
	defs := map[string]string{}

	c.FlagSet.VisitAll(func(f *flag.Flag) {
		fs.Var(f.Value, f.Name, f.Usage)
		defs[f.Name] = f.DefValue
	})
	fs.VisitAll(func(f *flag.Flag) {
		if def, ok := defs[f.Name]; ok {
			f.DefValue = def
		}
	})
}

func (c *BaseCommand) Run(ctx context.Context, fs *flag.FlagSet, args ...any) error {
	return fmt.Errorf("%w: %s %s", ErrUnimplementedCommand, c.Prefix, c.name)
}

var _ Command = (*BaseCommand)(nil)
