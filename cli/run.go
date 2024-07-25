package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/subcommands"
)

func CliContext() (context.Context, func()) {
	ctx := context.Background()

	ctx, cancel1 := context.WithCancel(ctx)

	ctx, cancel2 := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

	return ctx, func() {
		cancel2()
		cancel1()
	}
}

func Run(ctx context.Context, args []string, extras ...any) subcommands.ExitStatus {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	commander := subcommands.NewCommander(fs, args[0])

	arguments := DefaultGlobalArguments
	arguments.Setup(fs, commander)

	commander.Register(commander.HelpCommand(), "")
	commander.Register(commander.FlagsCommand(), "")
	commander.Register(commander.CommandsCommand(), "")

	commander.Register(NewRegistryCommand(args[0], &arguments), "")

	switch err := fs.Parse(args[1:]); {
	case errors.Is(err, flag.ErrHelp):
		return subcommands.ExitUsageError
	case err == nil:
		break
	default:
		fmt.Fprintf(commander.Error, "error: %s", err)
		return subcommands.ExitFailure
	}

	return commander.Execute(ctx, extras...)
}
