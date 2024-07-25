package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

var (
	ErrUsage = errors.New("Incorrect usage")
)

type Command interface {
	Name() string
	Usage() string
	Synopsis() string
	SetFlags(*flag.FlagSet)
	Run(context.Context, *flag.FlagSet, ...any) error
}

type commandWrapper struct {
	Command
}

func (w commandWrapper) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) subcommands.ExitStatus {
	switch err := w.Run(ctx, fs, args...); {
	case err == nil:
		return subcommands.ExitSuccess
	case errors.Is(err, flag.ErrHelp) || errors.Is(err, ErrUsage):
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitUsageError
	default:
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
}

type subcommand struct {
	subcommands.Command
	group string
}

type Subcommands struct {
	BaseCommand

	subcommands []subcommand
	commander   *subcommands.Commander
}

func NewSubcommands(prefix, name, usage string) Subcommands {
	s := Subcommands{
		BaseCommand: NewBaseCommand(prefix, name, usage),
	}
	s.commander = subcommands.NewCommander(s.BaseCommand.FlagSet, prefix+" "+name)

	return s
}

func (s *Subcommands) Name() string {
	return s.BaseCommand.Name()
}

func (s *Subcommands) Usage() string {
	return s.BaseCommand.Usage()
}

func (s *Subcommands) Synopsis() string {
	var b bytes.Buffer
	s.commander.Explain(&b)
	return b.String()
}

func (s *Subcommands) SetFlags(fs *flag.FlagSet) {
	s.commander = subcommands.NewCommander(fs, s.SubPrefix())

	s.commander.Register(s.commander.HelpCommand(), "")
	for _, subcommand := range s.subcommands {
		s.commander.Register(subcommand, subcommand.group)
	}

	s.BaseCommand.SetFlags(fs)
	s.BaseCommand.FlagSet.VisitAll(func(f *flag.Flag) {
		s.commander.ImportantFlag(f.Name)
	})
}

func (s *Subcommands) SubPrefix() string {
	return s.Prefix + " " + s.Name()
}

func (s *Subcommands) Register(cmd Command, group string) {
	s.RegisterCommand(commandWrapper{cmd}, group)
}

func (s *Subcommands) RegisterCommand(cmd subcommands.Command, group string) {
	s.subcommands = append(s.subcommands, subcommand{cmd, group})
	if s.commander != nil {
		s.commander.Register(cmd, group)
	}
}

func (s *Subcommands) Execute(ctx context.Context, fs *flag.FlagSet, args ...any) subcommands.ExitStatus {
	return s.commander.Execute(ctx, args...)
}
