package cli

import (
	"context"
	"encoding"
	"errors"
	"flag"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
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

type TextRepresentable interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

type TextVar struct {
	value TextRepresentable
}

func (tv TextVar) Set(value string) error {
	return tv.value.UnmarshalText([]byte(value))
}

func (tv TextVar) String() string {
	d, _ := tv.value.MarshalText()
	return string(d)
}

func (tv TextVar) Type() string {
	t := reflect.TypeOf(tv.value)
	if t.Kind() == reflect.Interface || t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return strings.TrimSuffix(t.Name(), "Value")
}

func Run(ctx context.Context, args []string, extras ...any) error {
	rootCmd := NewRootCommand()

	arguments := DefaultGlobalArguments
	arguments.Setup(rootCmd)

	rootCmd.AddCommand(
		NewRegistryCommand(&arguments),
		NewPackageCommand(&arguments),
		NewRepoCommand(&arguments),
	)

	rootCmd.SetArgs(args[1:])
	return rootCmd.ExecuteContext(ctx)
}

func ErrorToExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		return 2
	default:
		return 1
	}
}
