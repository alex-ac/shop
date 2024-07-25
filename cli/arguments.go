package cli

import (
	"flag"

	"github.com/alex-ac/shop"
	"github.com/google/subcommands"
)

type GlobalArguments struct {
	Config       string
	OutputFormat OutputFormat
}

var DefaultGlobalArguments = GlobalArguments{
	OutputFormat: DefaultOutputFormat,
}

func (a *GlobalArguments) Setup(fs *flag.FlagSet, commander *subcommands.Commander) {
	fs.StringVar(&a.Config, "config", a.Config, "Path to the config file to use.")
	fs.TextVar(&a.OutputFormat, "o", a.OutputFormat, "Output format.")

	commander.ImportantFlag("config")
	commander.ImportantFlag("o")
}

func (a *GlobalArguments) ResolveConfig() (err error) {
	if a.Config == "" {
		a.Config, err = shop.FindConfigFile()
	}

	return
}

func (a *GlobalArguments) LoadConfig() (cfg shop.Config, err error) {
	err = a.ResolveConfig()

	if err == nil {
		cfg, err = shop.LoadConfig(a.Config)
	}

	return
}

func (a *GlobalArguments) SaveConfig(cfg shop.Config) (err error) {
	err = a.ResolveConfig()

	if err == nil {
		err = shop.SaveConfig(cfg, a.Config)
	}

	return
}
